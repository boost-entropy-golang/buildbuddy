load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//enterprise:__subpackages__"])

go_library(
    name = "ociconv",
    srcs = ["ociconv.go"],
    importpath = "github.com/buildbuddy-io/buildbuddy/enterprise/server/util/ociconv",
    deps = [
        "//enterprise/server/util/ext4",
        "//enterprise/server/util/oci",
        "//server/util/disk",
        "//server/util/hash",
        "//server/util/log",
        "//server/util/status",
        "//server/util/tracing",
        "//third_party/singleflight",
        "@com_github_google_go_containerregistry//pkg/v1/mutate",
    ],
)

go_test(
    name = "ociconv_test",
    srcs = ["ociconv_test.go"],
    # TODO: remove skopeo dep and run on default pool.
    exec_properties = {
        "test.Pool": "bare",
        "test.use-self-hosted-executors": "true",
        "test.container-image": "none",
    },
    tags = ["bare"],
    target_compatible_with = ["@platforms//os:linux"],
    deps = [
        ":ociconv",
        "//enterprise/server/util/oci",
        "//server/testutil/testfs",
        "@com_github_stretchr_testify//require",
    ],
)
