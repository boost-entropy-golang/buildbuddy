load("@io_bazel_rules_go//go:def.bzl", "go_library")

go_library(
    name = "sidecar",
    srcs = ["sidecar.go"],
    importpath = "github.com/buildbuddy-io/buildbuddy/cli/sidecar",
    deps = [
        "//cli/arg",
        "//cli/config",
        "//cli/log",
        "//cli/storage",
        "//cli/version",
        "//cli/workspace",
        "//proto:sidecar_go_proto",
        "//server/util/grpc_client",
        "@com_github_google_shlex//:shlex",
    ],
)

package(default_visibility = ["//cli:__subpackages__"])
