load("@io_bazel_rules_go//go:def.bzl", "go_test")

package(default_visibility = ["//enterprise:__subpackages__"])

go_test(
    name = "ci_runner_test",
    size = "small",
    srcs = ["ci_runner_test.go"],
    data = [
        "//enterprise/server/cmd/ci_runner",
    ],
    # Run the ci_runner_test in the same environment that the CI runner uses in prod,
    # since we invoke the ci runner binary directly.
    # TODO(bduffany): Add an ubuntu 20.04-based CI runner image and use that
    # here.
    # exec_properties = {
    #     "container-image": "docker://gcr.io/flame-public/buildbuddy-ci-runner:v2.3.0",
    # },
    exec_properties = {
        # TODO: remove network dependency.
        "test.dockerNetwork": "bridge",
        # TODO: fix flakiness on OCI runner.
        "test.workload-isolation-type": "podman",
    },
    shard_count = 29,
    x_defs = {
        "ciRunnerRunfilePath": "$(rlocationpath //enterprise/server/cmd/ci_runner)",
    },
    deps = [
        "//proto:build_event_stream_go_proto",
        "//proto:command_line_go_proto",
        "//proto:eventlog_go_proto",
        "//proto:invocation_go_proto",
        "//proto:invocation_status_go_proto",
        "//proto:remote_execution_go_proto",
        "//proto:remote_execution_log_go_proto",
        "//server/remote_cache/cachetools",
        "//server/testutil/app",
        "//server/testutil/buildbuddy",
        "//server/testutil/testbazel",
        "//server/testutil/testfs",
        "//server/testutil/testgit",
        "//server/testutil/testshell",
        "@com_github_google_uuid//:uuid",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
        "@io_bazel_rules_go//go/runfiles:go_default_library",
        "@org_golang_google_protobuf//encoding/protodelim",
    ],
)
