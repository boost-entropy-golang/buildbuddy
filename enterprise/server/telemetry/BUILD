load("@io_bazel_rules_go//go:def.bzl", "go_library")

package(default_visibility = ["//enterprise:__subpackages__"])

go_library(
    name = "telemetry",
    srcs = ["telemetry_server.go"],
    importpath = "github.com/buildbuddy-io/buildbuddy/enterprise/server/telemetry",
    deps = [
        "//proto:telemetry_go_proto",
        "//server/environment",
        "//server/interfaces",
        "//server/rpc/interceptors",
        "//server/tables",
        "//server/util/grpc_server",
        "//server/util/log",
        "@io_gorm_gorm//clause",
        "@org_golang_google_genproto_googleapis_rpc//status",
        "@org_golang_google_grpc//:grpc",
        "@org_golang_google_grpc//codes",
    ],
)
