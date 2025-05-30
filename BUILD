load("@aspect_bazel_lib//lib:copy_to_bin.bzl", "copy_to_bin")
load("@aspect_rules_ts//ts:defs.bzl", "ts_config")
load("@bazel_gazelle//:def.bzl", "DEFAULT_LANGUAGES", "gazelle", "gazelle_binary")
load("@bazel_skylib//rules:write_file.bzl", "write_file")
load("@com_github_bazelbuild_buildtools//buildifier:def.bzl", "buildifier")
load("@com_github_sluongng_nogo_analyzer//staticcheck:def.bzl", "ANALYZERS", "staticcheck_analyzers")
load("@io_bazel_rules_go//go:def.bzl", "nogo")
load("@npm//:defs.bzl", "npm_link_all_packages")
load("//rules/go:index.bzl", "go_sdk_tool")

package(default_visibility = ["//visibility:public"])

npm_link_all_packages(name = "node_modules")

# Rendered JSON result could be checked by doing:
#   bazel build //:no_go_config
#   cat bazel-bin/no_go_config.json | jq .
write_file(
    name = "nogo_config",
    out = "nogo_config.json",
    content = [
        json.encode_indent(
            {
                "exhaustive": {
                    "exclude_files": {
                        "external[\\\\,\\/]": "third_party",
                    },
                    "analyzer_flags": {
                        "default-signifies-exhaustive": "true",
                    },
                },
            } | {
                analyzer: {
                    "exclude_files": {
                        "external[\\\\,\\/]": "third_party",
                        ".*\\.pb\\.go": "Auto-generated proto files",
                        # TODO(sluongng): this should be fixed on rules_go side
                        # https://github.com/bazelbuild/rules_go/issues/3619
                        "cgo[\\\\,\\/]github.com[\\\\,\\/]shirou[\\\\,\\/]gopsutil[\\\\,\\/]": "third_party cgo",
                    },
                }
                for analyzer in ANALYZERS + [
                    "appends",
                    "asmdecl",
                    "assign",
                    "atomicalign",
                    "bools",
                    "buildtag",
                    # "cgocall",
                    "composites",
                    "copylocks",
                    "deepequalerrors",
                    "defers",
                    "directive",
                    "errorsas",
                    # Noisy and is not part of 'go vet'
                    # "fieldalignment",
                    "framepointer",
                    "httpresponse",
                    "ifaceassert",
                    "loopclosure",
                    "lostcancel",
                    "nilfunc",
                    "nilness",
                    "printf",
                    # Everyone shadows `err`
                    # "shadow",
                    "shift",
                    "sigchanyzer",
                    "slog",
                    "sortslice",
                    "stdmethods",
                    "stdversion",
                    "stringintconv",
                    "structtag",
                    "testinggoroutine",
                    "tests",
                    "timeformat",
                    "unmarshal",
                    "unreachable",
                    "unsafeptr",
                    "unusedresult",
                    "unusedwrite",
                ]
            },
        ),
    ],
)

nogo(
    name = "vet",
    config = ":nogo_config.json",
    visibility = ["//visibility:public"],
    deps = [
        "@org_golang_x_tools//go/analysis/passes/appends",
        "@org_golang_x_tools//go/analysis/passes/asmdecl",
        "@org_golang_x_tools//go/analysis/passes/assign",
        "@org_golang_x_tools//go/analysis/passes/atomic",
        "@org_golang_x_tools//go/analysis/passes/atomicalign",
        "@org_golang_x_tools//go/analysis/passes/bools",
        "@org_golang_x_tools//go/analysis/passes/buildtag",
        # "@org_golang_x_tools//go/analysis/passes/cgocall",
        "@org_golang_x_tools//go/analysis/passes/composite",
        "@org_golang_x_tools//go/analysis/passes/copylock",
        "@org_golang_x_tools//go/analysis/passes/deepequalerrors",
        "@org_golang_x_tools//go/analysis/passes/defers",
        "@org_golang_x_tools//go/analysis/passes/directive",
        "@org_golang_x_tools//go/analysis/passes/errorsas",
        # "@org_golang_x_tools//go/analysis/passes/fieldalignment",
        "@org_golang_x_tools//go/analysis/passes/framepointer",
        "@org_golang_x_tools//go/analysis/passes/httpresponse",
        "@org_golang_x_tools//go/analysis/passes/ifaceassert",
        "@org_golang_x_tools//go/analysis/passes/loopclosure",
        "@org_golang_x_tools//go/analysis/passes/lostcancel",
        "@org_golang_x_tools//go/analysis/passes/nilfunc",
        "@org_golang_x_tools//go/analysis/passes/nilness",
        "@org_golang_x_tools//go/analysis/passes/printf",
        # Everyone shadows `err`
        # "@org_golang_x_tools//go/analysis/passes/shadow",
        "@org_golang_x_tools//go/analysis/passes/shift",
        "@org_golang_x_tools//go/analysis/passes/sigchanyzer",
        "@org_golang_x_tools//go/analysis/passes/slog",
        "@org_golang_x_tools//go/analysis/passes/sortslice",
        "@org_golang_x_tools//go/analysis/passes/stdmethods",
        "@org_golang_x_tools//go/analysis/passes/stdversion",
        "@org_golang_x_tools//go/analysis/passes/stringintconv",
        "@org_golang_x_tools//go/analysis/passes/structtag",
        "@org_golang_x_tools//go/analysis/passes/testinggoroutine",
        "@org_golang_x_tools//go/analysis/passes/tests",
        "@org_golang_x_tools//go/analysis/passes/timeformat",
        "@org_golang_x_tools//go/analysis/passes/unmarshal",
        "@org_golang_x_tools//go/analysis/passes/unreachable",
        "@org_golang_x_tools//go/analysis/passes/unsafeptr",
        "@org_golang_x_tools//go/analysis/passes/unusedresult",
        "@org_golang_x_tools//go/analysis/passes/unusedwrite",
        "@com_github_nishanths_exhaustive//:exhaustive",
    ] + staticcheck_analyzers(ANALYZERS + [
        "-SA1019",
        "-SA1029",
        "-ST1000",
        "-ST1003",
        "-ST1005",
        "-ST1006",
        "-ST1008",
        "-ST1012",
        "-ST1016",
        "-ST1017",
        "-ST1020",
        "-ST1021",
        "-ST1022",
        "-ST1023",
        "-QF1001",
        "-QF1003",
        "-QF1004",
        "-QF1005",
        "-QF1006",
        "-QF1008",
        "-QF1011",
        "-QF1012",
        "-U1000",
    ]),
)

gazelle_binary(
    name = "bb_gazelle_binary",
    languages = DEFAULT_LANGUAGES + ["@bazel_gazelle//language/bazel/visibility"],
)

# Ignore the node_modules dir
# gazelle:exclude node_modules
# Ignore generated proto files
# gazelle:exclude **/*.pb.go
# gazelle:exclude bundle.go
# gazelle:exclude enterprise/bundle.go
# Prefer generated BUILD files to be called BUILD over BUILD.bazel
# gazelle:build_file_name BUILD,BUILD.bazel
# gazelle:prefix github.com/buildbuddy-io/buildbuddy
# gazelle:proto disable
# gazelle:map_kind ts_project ts_library //rules/typescript:index.bzl
# gazelle:exclude **/node_modules/**
# TODO(siggisim): remove once we support .css imports properly
# gazelle:exclude website/**
# gazelle:resolve go kythe.io/kythe/proto/common_go_proto @io_kythe//kythe/proto:common_go_proto
# gazelle:resolve go kythe.io/kythe/proto/filetree_go_proto @io_kythe//kythe/proto:filetree_go_proto
# gazelle:resolve go kythe.io/kythe/proto/graph_go_proto @io_kythe//kythe/proto:graph_go_proto
# gazelle:resolve go kythe.io/kythe/proto/xref_go_proto @io_kythe//kythe/proto:xref_go_proto
#
# This is a list of default when using Gazelle from BzlMod.
# We force these mapping manually so that we do not oscillate during migrating to BzlMod
# (and potentially any revert back to WORKSPACE mode).
#
# TODO(sluongng): remove these once we deem BzlMod stable enough
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/config @bazel_gazelle//config
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/label @bazel_gazelle//label
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/language @bazel_gazelle//language
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/language/bazel/visibility @bazel_gazelle//language/bazel/visibility
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/language/go @bazel_gazelle//language/go
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/language/proto @bazel_gazelle//language/proto
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/repo @bazel_gazelle//repo
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/resolve @bazel_gazelle//resolve
# gazelle:resolve go github.com/bazelbuild/bazel-gazelle/rule @bazel_gazelle//rule
# gazelle:resolve go github.com/bazelbuild/rules_go/go/runfiles @io_bazel_rules_go//go/runfiles
# gazelle:resolve go github.com/bazelbuild/rules_go/go/tools/bazel @io_bazel_rules_go//go/tools/bazel
#
# Make these the default compilers for proto rules.
# See https://github.com/bazelbuild/rules_go/pull/3761 for more details
# gazelle:go_proto_compilers	@io_bazel_rules_go//proto:go_proto,@io_bazel_rules_go//proto:go_grpc_v2
gazelle(
    name = "gazelle",
    gazelle = ":bb_gazelle_binary",
)

buildifier(
    name = "buildifier",
)

go_sdk_tool(
    name = "go",
    goroot_relative_path = "bin/go",
)

# Example usage: "bazel run //:gofmt -- -w ."
go_sdk_tool(
    name = "gofmt",
    goroot_relative_path = "bin/gofmt",
)

exports_files([
    "package.json",
    "yarn.lock",
])

copy_to_bin(
    name = "swcrc",
    srcs = [".swcrc"],
)

ts_config(
    name = "tsconfig",
    src = ":tsconfig.json",
)

config_setting(
    name = "fastbuild",
    values = {
        "compilation_mode": "fastbuild",
    },
)

config_setting(
    name = "release_build",
    values = {"define": "release=true"},
)

config_setting(
    name = "static",
    flag_values = {"@io_bazel_rules_go//go/config:static": "true"},
)

package_group(
    name = "os",
    packages = [
        "//app/...",
        "//config/...",
        "//deployment/...",
        "//docs/...",
        "//node_modules/...",
        "//proto/...",
        "//rules/...",
        "//server/...",
        "//static/...",
        "//templates/...",
        "//tools/...",
    ],
)

package_group(
    name = "enterprise",
    packages = [
        "//enterprise/...",
    ],
)

platform(
    name = "firecracker",
    constraint_values = [
        "@platforms//cpu:x86_64",
        "@platforms//os:linux",
    ],
    exec_properties = {
        "workload-isolation-type": "firecracker",
    },
)

platform(
    name = "firecracker_vfs",
    constraint_values = [
        "@platforms//cpu:x86_64",
        "@platforms//os:linux",
    ],
    exec_properties = {
        "workload-isolation-type": "firecracker",
        "enable-vfs": "true",
    },
)

platform(
    name = "vfs",
    constraint_values = [
        "@platforms//cpu:x86_64",
        "@platforms//os:linux",
    ],
    exec_properties = {
        "enable-vfs": "true",
    },
)
