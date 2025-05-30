package digest

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"fmt"
	"hash"
	"io"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"

	"github.com/buildbuddy-io/buildbuddy/server/util/alert"
	"github.com/buildbuddy-io/buildbuddy/server/util/log"
	"github.com/buildbuddy-io/buildbuddy/server/util/proto"
	"github.com/buildbuddy-io/buildbuddy/server/util/status"
	"github.com/zeebo/blake3"
	"google.golang.org/genproto/googleapis/rpc/errdetails"

	repb "github.com/buildbuddy-io/buildbuddy/proto/remote_execution"
	rspb "github.com/buildbuddy-io/buildbuddy/proto/resource"
	guuid "github.com/google/uuid"
	gcodes "google.golang.org/grpc/codes"
	gmetadata "google.golang.org/grpc/metadata"
	gstatus "google.golang.org/grpc/status"
)

const (
	EmptySha256 = "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	EmptyHash   = ""
)

var (
	uploadRegex      *regexp.Regexp
	downloadRegex    *regexp.Regexp
	actionCacheRegex *regexp.Regexp

	knownDigestFunctions = []repb.DigestFunction_Value{
		repb.DigestFunction_SHA256,
		repb.DigestFunction_SHA384,
		repb.DigestFunction_SHA512,
		repb.DigestFunction_SHA1,
		repb.DigestFunction_BLAKE3,
	}
)

func init() {
	hashMatchers := make([]string, 0)
	for _, df := range knownDigestFunctions {
		hashMatchers = append(hashMatchers, fmt.Sprintf("[a-f0-9]{%d}", hashLength(df)))
	}
	joinedMatchers := strings.Join(hashMatchers, "|")

	// Matches:
	// - "blobs/469db13020c60f8bdf9c89aa4e9a449914db23139b53a24d064f967a51057868/39120"
	// - "blobs/ac/469db13020c60f8bdf9c89aa4e9a449914db23139b53a24d064f967a51057868/39120"
	// - "uploads/2042a8f9-eade-4271-ae58-f5f6f5a32555/blobs/8afb02ca7aace3ae5cd8748ac589e2e33022b1a4bfd22d5d234c5887e270fe9c/17997850"
	uploadRegex = regexp.MustCompile(fmt.Sprintf(`^(?:(?:(?P<instance_name>.*)/)?uploads/(?P<uuid>[a-f0-9-]{36})/)?(?P<blob_type>blobs|compressed-blobs/zstd)/(?:(?P<digest_function>blake3)/)?(?P<hash>%s)/(?P<size>\d+)`, joinedMatchers))
	downloadRegex = regexp.MustCompile(fmt.Sprintf(`^(?:(?P<instance_name>.*)/)?(?P<blob_type>blobs|compressed-blobs/zstd)/(?:(?P<digest_function>blake3)/)?(?P<hash>%s)/(?P<size>\d+)`, joinedMatchers))
	actionCacheRegex = regexp.MustCompile(fmt.Sprintf(`^(?:(?P<instance_name>.*)/)?(?P<blob_type>blobs|compressed-blobs/zstd)/ac/(?:(?P<digest_function>blake3)/)?(?P<hash>%s)/(?P<size>\d+)`, joinedMatchers))
}

func SupportedDigestFunctions() []repb.DigestFunction_Value {
	return knownDigestFunctions
}

type ResourceName struct {
	rn *rspb.ResourceName
}

// Prefer either CASResourceNameFromProto or ACResourceNameFromProto.
func ResourceNameFromProto(in *rspb.ResourceName) *ResourceName {
	rn := in.CloneVT()
	// TODO(tylerw): remove once digest function is explicit everywhere.
	if rn.GetDigestFunction() == repb.DigestFunction_UNKNOWN {
		rn.DigestFunction = repb.DigestFunction_SHA256
	}
	return &ResourceName{
		rn: rn,
	}
}

func CASResourceNameFromProto(in *rspb.ResourceName) (*CASResourceName, error) {
	if in.GetCacheType() != rspb.CacheType_CAS {
		return nil, status.FailedPreconditionErrorf("ResourceName is not a CAS resource name: %s", in)
	}
	return &CASResourceName{*ResourceNameFromProto(in)}, nil
}

func ACResourceNameFromProto(in *rspb.ResourceName) (*ACResourceName, error) {
	if in.GetCacheType() != rspb.CacheType_AC {
		return nil, status.FailedPreconditionErrorf("ResourceName is not an AC resource name: %s", in)
	}
	return &ACResourceName{*ResourceNameFromProto(in)}, nil
}

// Prefer either NewCASResourceName or NewACResourceName.
func NewResourceName(d *repb.Digest, instanceName string, cacheType rspb.CacheType, digestFunction repb.DigestFunction_Value) *ResourceName {
	if digestFunction == repb.DigestFunction_UNKNOWN {
		digestFunction = InferOldStyleDigestFunctionInDesperation(d)
	}
	return &ResourceName{
		rn: &rspb.ResourceName{
			Digest:         d,
			InstanceName:   instanceName,
			Compressor:     repb.Compressor_IDENTITY,
			CacheType:      cacheType,
			DigestFunction: digestFunction,
		},
	}
}

func NewCASResourceName(d *repb.Digest, instanceName string, digestFunction repb.DigestFunction_Value) *CASResourceName {
	return &CASResourceName{*NewResourceName(d, instanceName, rspb.CacheType_CAS, digestFunction)}
}

func NewACResourceName(d *repb.Digest, instanceName string, digestFunction repb.DigestFunction_Value) *ACResourceName {
	return &ACResourceName{*NewResourceName(d, instanceName, rspb.CacheType_AC, digestFunction)}
}

func (r *ResourceName) CheckCAS() (*CASResourceName, error) {
	if r.rn.GetCacheType() != rspb.CacheType_CAS {
		return nil, status.FailedPreconditionErrorf("ResourceName is not a CAS resource name: %s", r.rn)
	}
	return &CASResourceName{*r}, nil
}

func (r *ResourceName) CheckAC() (*ACResourceName, error) {
	if r.rn.GetCacheType() != rspb.CacheType_AC {
		return nil, status.FailedPreconditionErrorf("ResourceName is not an AC resource name: %s", r.rn)
	}
	return &ACResourceName{*r}, nil
}

func (r *ResourceName) ToProto() *rspb.ResourceName {
	return r.rn
}

func (r *ResourceName) GetDigest() *repb.Digest {
	return r.rn.GetDigest()
}

func (r *ResourceName) GetDigestFunction() repb.DigestFunction_Value {
	return r.rn.GetDigestFunction()
}

func (r *ResourceName) GetInstanceName() string {
	return r.rn.GetInstanceName()
}

func (r *ResourceName) GetCacheType() rspb.CacheType {
	return r.rn.GetCacheType()
}

func (r *ResourceName) GetCompressor() repb.Compressor_Value {
	return r.rn.GetCompressor()
}

func (r *ResourceName) SetCompressor(compressor repb.Compressor_Value) {
	r.rn.Compressor = compressor
}

func (r *ResourceName) IsEmpty() bool {
	return IsEmptyHash(r.rn.GetDigest(), r.rn.GetDigestFunction())
}

func isLowerHex(s string) bool {
	for _, ch := range s {
		if (ch < '0' || ch > '9') && (ch < 'a' || ch > 'f') {
			return false
		}
	}
	return true
}

func (r *ResourceName) Validate() error {
	d := r.rn.GetDigest()
	if d == nil {
		return status.InvalidArgumentError("Invalid (nil) Digest")
	}
	if d.GetSizeBytes() < 0 {
		return status.InvalidArgumentErrorf("Invalid (negative) digest size")
	}
	if d.GetSizeBytes() == int64(0) {
		if r.IsEmpty() {
			return nil
		}
		return status.InvalidArgumentError("Invalid (zero-length) SHA256 hash")
	}
	hash := d.GetHash()
	if expected := hashLength(r.GetDigestFunction()); len(hash) != expected {
		return status.InvalidArgumentErrorf("Invalid length hash. Expected len %v for %s function. Got %v",
			expected, r.GetDigestFunction().String(), len(hash))
	}
	if !isLowerHex(hash) {
		return status.InvalidArgumentError("Hash isn't all lower case hex characters.")
	}
	return nil
}

type CASResourceName struct {
	ResourceName
}

// DownloadString returns a string representing the resource name for download
// purposes.
func (r *CASResourceName) DownloadString() string {
	// Normalize slashes, e.g. "//foo/bar//"" becomes "/foo/bar".
	instanceName := filepath.Join(filepath.SplitList(r.GetInstanceName())...)
	if isOldStyleDigestFunction(r.rn.DigestFunction) {
		return fmt.Sprintf(
			"%s/%s/%s/%d",
			instanceName, blobTypeSegment(r.GetCompressor()),
			r.GetDigest().GetHash(), r.GetDigest().GetSizeBytes())
	} else {
		return fmt.Sprintf(
			"%s/%s/%s/%s/%d",
			instanceName, blobTypeSegment(r.GetCompressor()),
			strings.ToLower(r.rn.DigestFunction.String()),
			r.GetDigest().GetHash(), r.GetDigest().GetSizeBytes())
	}
}

// NewUploadString returns a new string representing the resource name for
// upload purposes each time it is called.
func (r *CASResourceName) NewUploadString() string {
	// Normalize slashes, e.g. "//foo/bar//"" becomes "/foo/bar".
	instanceName := filepath.Join(filepath.SplitList(r.GetInstanceName())...)
	u := guuid.New()
	if isOldStyleDigestFunction(r.rn.DigestFunction) {
		return fmt.Sprintf(
			"%s/uploads/%s/%s/%s/%d",
			instanceName, u.String(), blobTypeSegment(r.GetCompressor()),
			r.GetDigest().GetHash(), r.GetDigest().GetSizeBytes(),
		)
	} else {
		return fmt.Sprintf(
			"%s/uploads/%s/%s/%s/%s/%d",
			instanceName, u.String(), blobTypeSegment(r.GetCompressor()),
			strings.ToLower(r.rn.DigestFunction.String()),
			r.GetDigest().GetHash(), r.GetDigest().GetSizeBytes(),
		)
	}
}

type ACResourceName struct {
	ResourceName
}

// ActionCacheString returns a string representing the resource name for in
// the action cache. This is BuildBuddy specific.
func (r *ACResourceName) ActionCacheString() string {
	// Normalize slashes, e.g. "//foo/bar//"" becomes "/foo/bar".
	instanceName := filepath.Join(filepath.SplitList(r.GetInstanceName())...)
	if isOldStyleDigestFunction(r.rn.DigestFunction) {
		return fmt.Sprintf(
			"%s/%s/ac/%s/%d",
			instanceName, blobTypeSegment(r.GetCompressor()),
			r.GetDigest().GetHash(), r.GetDigest().GetSizeBytes())
	} else {
		return fmt.Sprintf(
			"%s/%s/ac/%s/%s/%d",
			instanceName, blobTypeSegment(r.GetCompressor()),
			strings.ToLower(r.rn.DigestFunction.String()),
			r.GetDigest().GetHash(), r.GetDigest().GetSizeBytes())
	}
}

func CacheTypeToPrefix(cacheType rspb.CacheType) string {
	switch cacheType {
	case rspb.CacheType_CAS:
		return ""
	case rspb.CacheType_AC:
		return "ac"
	default:
		alert.UnexpectedEvent("unknown_cache_type", "type: %v", cacheType)
		return "unknown"
	}
}

func ResourceNames(cacheType rspb.CacheType, remoteInstanceName string, digests []*repb.Digest) []*rspb.ResourceName {
	rns := make([]*rspb.ResourceName, 0, len(digests))
	for _, d := range digests {
		rns = append(rns, &rspb.ResourceName{
			Digest:       d,
			InstanceName: remoteInstanceName,
			Compressor:   repb.Compressor_IDENTITY,
			CacheType:    cacheType,
		})
	}
	return rns
}

func ResourceNameMap(cacheType rspb.CacheType, remoteInstanceName string, digestMap map[*repb.Digest][]byte) map[*rspb.ResourceName][]byte {
	rnMap := make(map[*rspb.ResourceName][]byte, len(digestMap))
	for d, data := range digestMap {
		rn := &rspb.ResourceName{
			Digest:       d,
			InstanceName: remoteInstanceName,
			Compressor:   repb.Compressor_IDENTITY,
			CacheType:    cacheType,
		}
		rnMap[rn] = data
	}
	return rnMap
}

// Key is a representation of a digest that can be used as a map key.
type Key struct {
	Hash      string
	SizeBytes int64
}

func NewKey(digest *repb.Digest) Key {
	return Key{Hash: digest.GetHash(), SizeBytes: digest.GetSizeBytes()}
}
func (dk Key) ToDigest() *repb.Digest {
	return &repb.Digest{Hash: dk.Hash, SizeBytes: dk.SizeBytes}
}

func ComputeForMessage(in proto.Message, digestType repb.DigestFunction_Value) (*repb.Digest, error) {
	data, err := proto.Marshal(in)
	if err != nil {
		return nil, err
	}
	return Compute(bytes.NewReader(data), digestType)
}

func HashForDigestType(digestType repb.DigestFunction_Value) (hash.Hash, error) {
	switch digestType {
	case repb.DigestFunction_SHA1:
		return sha1.New(), nil
	case repb.DigestFunction_SHA256:
		return sha256.New(), nil
	case repb.DigestFunction_SHA384:
		return sha512.New384(), nil
	case repb.DigestFunction_SHA512:
		return sha512.New(), nil
	case repb.DigestFunction_BLAKE3:
		return blake3.New(), nil
	case repb.DigestFunction_UNKNOWN:
		// TODO(tylerw): make this a warning when clients support this.
		// log.Warningf("Digest function was unset: defaulting to SHA256")
		return sha256.New(), nil
	default:
		return nil, status.UnimplementedErrorf("No support for digest type: %s", digestType)
	}
}

func InferOldStyleDigestFunctionInDesperation(d *repb.Digest) repb.DigestFunction_Value {
	switch len(d.GetHash()) {
	case sha1.Size * 2:
		return repb.DigestFunction_SHA1
	case md5.Size * 2:
		return repb.DigestFunction_MD5
	case sha256.Size * 2:
		return repb.DigestFunction_SHA256
	case sha512.Size384 * 2:
		return repb.DigestFunction_SHA384
	case sha512.Size * 2:
		return repb.DigestFunction_SHA512
	default:
		return repb.DigestFunction_UNKNOWN
	}
}

func IsEmptyHash(d *repb.Digest, digestFunction repb.DigestFunction_Value) bool {
	switch digestFunction {
	case repb.DigestFunction_SHA1:
		return d.GetHash() == "da39a3ee5e6b4b0d3255bfef95601890afd80709"
	case repb.DigestFunction_MD5:
		return d.GetHash() == "d41d8cd98f00b204e9800998ecf8427e"
	case repb.DigestFunction_SHA256:
		return d.GetHash() == "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	case repb.DigestFunction_SHA384:
		return d.GetHash() == "38b060a751ac96384cd9327eb1b1e36a21fdb71114be07434c0cc7bf63f6e1da274edebfe76f65fbd51ad2f14898b95b"
	case repb.DigestFunction_SHA512:
		return d.GetHash() == "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e"
	case repb.DigestFunction_BLAKE3:
		return d.GetHash() == "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262"
	default:
		return false
	}
}

func hashLength(digestFunction repb.DigestFunction_Value) int {
	switch digestFunction {
	case repb.DigestFunction_BLAKE3:
		return 32 * 2
	case repb.DigestFunction_SHA256:
		return sha256.Size * 2
	case repb.DigestFunction_SHA384:
		return sha512.Size384 * 2
	case repb.DigestFunction_SHA512:
		return sha512.Size * 2
	case repb.DigestFunction_SHA1:
		return sha1.Size * 2
	case repb.DigestFunction_MD5:
		return md5.Size * 2
	default:
		return -1
	}
}

func isOldStyleDigestFunction(digestFunction repb.DigestFunction_Value) bool {
	switch digestFunction {
	case repb.DigestFunction_SHA1:
		return true
	case repb.DigestFunction_MD5:
		return true
	case repb.DigestFunction_SHA256:
		return true
	case repb.DigestFunction_SHA384:
		return true
	case repb.DigestFunction_SHA512:
		return true
	default:
		return false
	}
}

func Compute(in io.Reader, digestType repb.DigestFunction_Value) (*repb.Digest, error) {
	h, err := HashForDigestType(digestType)
	if err != nil {
		return nil, err
	}

	// Read file in 32KB chunks (default)
	n, err := io.Copy(h, in)
	if err != nil {
		return nil, err
	}
	return &repb.Digest{
		Hash:      fmt.Sprintf("%x", h.Sum(nil)),
		SizeBytes: n,
	}, nil
}

func ComputeForFile(path string, digestType repb.DigestFunction_Value) (*repb.Digest, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Compute(f, digestType)
}

func isResourceName(url string, matcher *regexp.Regexp) bool {
	return matcher.MatchString(url)
}

func parseResourceName(resourceName string, matcher *regexp.Regexp, cacheType rspb.CacheType) (*ResourceName, error) {
	match := matcher.FindStringSubmatch(resourceName)
	result := make(map[string]string, len(match))
	for i, name := range matcher.SubexpNames() {
		if i != 0 && name != "" && i < len(match) {
			result[name] = match[i]
		}
	}
	hash, hashOK := result["hash"]
	sizeStr, sizeOK := result["size"]
	if !hashOK || !sizeOK {
		return nil, status.InvalidArgumentErrorf("Unparsable resource name: %s", resourceName)
	}
	if hash == "" {
		return nil, status.InvalidArgumentErrorf("Unparsable resource name (empty hash?): %s", resourceName)
	}
	sizeBytes, err := strconv.ParseInt(sizeStr, 10, 0)
	if err != nil {
		return nil, err
	}

	// Set the instance name, if one was present.
	instanceName := ""
	if in, ok := result["instance_name"]; ok {
		instanceName = in
	}

	// Determine compression level from blob type segment
	blobTypeStr, sizeOK := result["blob_type"]
	if !sizeOK {
		// Should never happen since the regex would not match otherwise.
		return nil, status.InvalidArgumentError(`Unparsable resource name: "/blobs" or "/compressed-blobs/zstd" missing or out of place`)
	}
	compressor := repb.Compressor_IDENTITY
	if blobTypeStr == "compressed-blobs/zstd" {
		compressor = repb.Compressor_ZSTD
	}
	d := &repb.Digest{Hash: hash, SizeBytes: sizeBytes}

	// Determine the digest function by looking at the digest length.
	// If a digest_function value was specified in the bytestream URL, this
	// is a new style hash, so lookup the type based on that value.
	digestFunction := InferOldStyleDigestFunctionInDesperation(d)
	if dfString, ok := result["digest_function"]; ok && dfString != "" {
		df, err := ParseFunction(dfString)
		if err != nil {
			return nil, err
		}
		digestFunction = df
	}
	r := NewResourceName(d, instanceName, cacheType, digestFunction)
	r.SetCompressor(compressor)
	return r, nil
}

func ParseUploadResourceName(resourceName string) (*CASResourceName, error) {
	rn, err := parseResourceName(resourceName, uploadRegex, rspb.CacheType_CAS)
	if err != nil {
		return nil, err
	}
	return rn.CheckCAS()
}

func ParseDownloadResourceName(resourceName string) (*CASResourceName, error) {
	rn, err := parseResourceName(resourceName, downloadRegex, rspb.CacheType_CAS)
	if err != nil {
		return nil, err
	}
	return rn.CheckCAS()
}

func ParseActionCacheResourceName(resourceName string) (*ACResourceName, error) {
	rn, err := parseResourceName(resourceName, actionCacheRegex, rspb.CacheType_AC)
	if err != nil {
		return nil, err
	}
	return rn.CheckAC()
}

func IsDownloadResourceName(url string) bool {
	return isResourceName(url, downloadRegex)
}

func IsActionCacheResourceName(url string) bool {
	return isResourceName(url, actionCacheRegex)
}

func blobTypeSegment(compressor repb.Compressor_Value) string {
	if compressor == repb.Compressor_ZSTD {
		return "compressed-blobs/zstd"
	}
	return "blobs"
}

func IsCacheDebuggingEnabled(ctx context.Context) bool {
	if hdrs := gmetadata.ValueFromIncomingContext(ctx, "debug-cache-hits"); len(hdrs) > 0 {
		if strings.ToLower(strings.TrimSpace(hdrs[0])) == "true" {
			return true
		}
	}
	return false
}

func MissingDigestError(d *repb.Digest) error {
	if d == nil {
		log.Infof("MissingDigestError called with nil digest. Stack trace:\n%s", string(debug.Stack()))
	}

	pf := &errdetails.PreconditionFailure{}
	pf.Violations = append(pf.Violations, &errdetails.PreconditionFailure_Violation{
		Type:    "MISSING",
		Subject: fmt.Sprintf("blobs/%s/%d", d.GetHash(), d.GetSizeBytes()),
	})
	st := gstatus.Newf(gcodes.FailedPrecondition, "Digest %v not found", d)
	if st, err := st.WithDetails(pf); err != nil {
		return status.InternalErrorf("Digest %v not found.", d)
	} else {
		return st.Err()
	}
}

// String returns the digest formatted as "HASH/SIZE" or the string "<nil>"
// if the digest is nil.
//
// Note: this is intended mainly for logging - to get a representation of a
// digest suitable for use as a map key, use NewKey instead.
func String(d *repb.Digest) string {
	if d == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%s/%d", d.Hash, d.SizeBytes)
}

// ElementsMatch returns whether two slices contain the same digests, ignoring the order of the elements.
// If there are duplicate elements, the number of appearances of each of them in both lists should match.
func ElementsMatch(s1 []*repb.Digest, s2 []*repb.Digest) bool {
	if len(s1) != len(s2) {
		return false
	}

	foundS1 := make(map[*repb.Digest]int, len(s1))
	for _, d := range s1 {
		foundS1[d]++
	}

	for _, d := range s2 {
		var numOccur int
		var inS1 bool
		if numOccur, inS1 = foundS1[d]; !inS1 {
			return false
		}
		if numOccur == 1 {
			delete(foundS1, d)
		} else {
			foundS1[d]--
		}
	}

	return true
}

// Diff returns the differences between two slices of digests. If the slices differ in the count of a non-unique element,
// that does not count as a difference
//
// missingFromS1 contains the digests that are in S2 but not S1
// missingFromS2 contains the digests that are in S1 but not S2
func Diff(s1 []*repb.Digest, s2 []*repb.Digest) (missingFromS1 []*repb.Digest, missingFromS2 []*repb.Digest) {
	missingFromS1 = make([]*repb.Digest, 0)
	missingFromS2 = make([]*repb.Digest, 0)

	s1Set := make(map[Key]struct{}, len(s1))
	for _, d := range s1 {
		s1Set[NewKey(d)] = struct{}{}
	}

	s2Set := make(map[Key]struct{}, len(s2))
	for _, d := range s2 {
		k := NewKey(d)
		s2Set[k] = struct{}{}

		if _, inS1 := s1Set[k]; !inS1 {
			missingFromS1 = append(missingFromS1, k.ToDigest())
		}
	}

	for k := range s1Set {
		if _, inS2 := s2Set[k]; !inS2 {
			missingFromS2 = append(missingFromS2, k.ToDigest())
		}
	}

	return missingFromS1, missingFromS2
}

type randomDataMaker struct {
	src              rand.Source
	compressionRatio float64
	val              int64
}

func (r *randomDataMaker) Read(p []byte) (n int, err error) {
	todo := len(p)
	offset := 0
	for {
		// Generate a new random int64 (8 bytes) if we haven't generated one
		// yet, or with a percent chance given by the compression ratio. This is
		// a *very* rough way to generate blobs with the average compression
		// ratios that we see in practice.
		if r.val == 0 || r.src.Int63()%100 >= int64(r.compressionRatio*100) {
			r.val = int64(r.src.Int63())
		}
		val := r.val
		for i := 0; i < 8; i++ {
			p[offset] = byte(val & 0xff)
			todo--
			if todo == 0 {
				return len(p), nil
			}
			offset++
			val >>= 8
		}
	}
}

type Generator struct {
	randMaker *randomDataMaker
	mu        sync.Mutex
}

// RandomGenerator returns a digest sample generator for use in testing tools.
// It generates digests with compression ratios similar to what we see in
// practice.
func RandomGenerator(seed int64) *Generator {
	return &Generator{
		randMaker: &randomDataMaker{
			src:              rand.NewSource(seed),
			compressionRatio: 0.7,
		},
	}
}

// UniformRandomGenerator generates uniformly random, incompressible digests,
// for use in testing. The data generated does not look realistic, but it is
// useful in cases where unique digests are needed.
func UniformRandomGenerator(seed int64) *Generator {
	return &Generator{
		randMaker: &randomDataMaker{
			src: rand.NewSource(seed),
		},
	}
}

func (g *Generator) RandomDigestReader(sizeBytes int64) (*repb.Digest, io.ReadSeeker, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Read some random bytes.
	buf := new(bytes.Buffer)
	if _, err := io.CopyN(buf, g.randMaker, sizeBytes); err != nil {
		return nil, nil, err
	}
	readSeeker := bytes.NewReader(buf.Bytes())

	// Compute a digest for the random bytes.
	d, err := Compute(readSeeker, repb.DigestFunction_SHA256)
	if err != nil {
		return nil, nil, err
	}
	if _, err := readSeeker.Seek(0, 0); err != nil {
		return nil, nil, err
	}
	return d, readSeeker, nil
}

func (g *Generator) RandomDigestBuf(sizeBytes int64) (*repb.Digest, []byte, error) {
	d, rs, err := g.RandomDigestReader(sizeBytes)
	if err != nil {
		return nil, nil, err
	}
	buf, err := io.ReadAll(rs)
	if err != nil {
		return nil, nil, err
	}
	return d, buf, nil
}

// ParseFunction parses a digest function name to a proto.
func ParseFunction(s string) (repb.DigestFunction_Value, error) {
	if df, ok := repb.DigestFunction_Value_value[strings.ToUpper(s)]; ok {
		return repb.DigestFunction_Value(df), nil
	}
	return 0, status.InvalidArgumentErrorf("unknown digest function %q", s)
}
