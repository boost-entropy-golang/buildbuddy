load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//enterprise:__subpackages__"])

go_library(
    name = "rangelease",
    srcs = ["rangelease.go"],
    importpath = "github.com/buildbuddy-io/buildbuddy/enterprise/server/raft/rangelease",
    deps = [
        "//enterprise/server/raft/client",
        "//enterprise/server/raft/constants",
        "//enterprise/server/raft/nodeliveness",
        "//enterprise/server/raft/rbuilder",
        "//enterprise/server/raft/replica",
        "//proto:raft_go_proto",
        "//server/util/log",
        "//server/util/proto",
        "//server/util/rangemap",
        "//server/util/status",
        "//server/util/tracing",
        "@io_opentelemetry_go_otel//attribute",
    ],
)

go_test(
    name = "rangelease_test",
    size = "small",
    srcs = ["rangelease_test.go"],
    deps = [
        ":rangelease",
        "//enterprise/server/raft/client",
        "//enterprise/server/raft/constants",
        "//enterprise/server/raft/keys",
        "//enterprise/server/raft/nodeliveness",
        "//enterprise/server/raft/replica",
        "//enterprise/server/raft/sender",
        "//enterprise/server/raft/testutil",
        "//proto:raft_go_proto",
        "//server/testutil/quarantine",
        "//server/util/log",
        "//server/util/proto",
        "//server/util/random",
        "//server/util/status",
        "@com_github_stretchr_testify//require",
    ],
)
