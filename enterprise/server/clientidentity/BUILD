load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//enterprise:__subpackages__"])

go_library(
    name = "clientidentity",
    srcs = ["clientidentity.go"],
    importpath = "github.com/buildbuddy-io/buildbuddy/enterprise/server/clientidentity",
    deps = [
        "//server/interfaces",
        "//server/real_environment",
        "//server/util/authutil",
        "//server/util/flag",
        "//server/util/status",
        "@com_github_golang_jwt_jwt_v4//:jwt",
        "@com_github_jonboulle_clockwork//:clockwork",
        "@org_golang_google_grpc//metadata",
    ],
)

go_test(
    name = "clientidentity_test",
    srcs = ["clientidentity_test.go"],
    deps = [
        ":clientidentity",
        "//server/interfaces",
        "//server/util/authutil",
        "//server/util/random",
        "//server/util/status",
        "//server/util/testing/flags",
        "@com_github_golang_jwt_jwt_v4//:jwt",
        "@com_github_jonboulle_clockwork//:clockwork",
        "@com_github_stretchr_testify//require",
        "@org_golang_google_grpc//metadata",
    ],
)
