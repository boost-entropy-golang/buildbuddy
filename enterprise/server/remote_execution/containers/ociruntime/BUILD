load("@io_bazel_rules_go//go:def.bzl", "go_library", "go_test")

package(default_visibility = ["//enterprise:__subpackages__"])

go_library(
    name = "ociruntime",
    srcs = ["ociruntime.go"],
    embedsrcs = [
        "hosts",
        "seccomp.json",
    ],
    importpath = "github.com/buildbuddy-io/buildbuddy/enterprise/server/remote_execution/containers/ociruntime",
    target_compatible_with = ["@platforms//os:linux"],
    visibility = ["//visibility:public"],
    deps = [
        "//enterprise/server/remote_execution/block_io",
        "//enterprise/server/remote_execution/cgroup",
        "//enterprise/server/remote_execution/commandutil",
        "//enterprise/server/remote_execution/container",
        "//enterprise/server/util/oci",
        "//proto:remote_execution_go_proto",
        "//proto:scheduler_go_proto",
        "//server/environment",
        "//server/interfaces",
        "//server/util/disk",
        "//server/util/flag",
        "//server/util/hash",
        "//server/util/log",
        "//server/util/networking",
        "//server/util/proto",
        "//server/util/status",
        "//server/util/unixcred",
        "//server/util/uuid",
        "//third_party/singleflight",
        "@com_github_google_go_containerregistry//pkg/v1:pkg",
        "@com_github_opencontainers_runtime_spec//specs-go",
        "@org_golang_x_sync//errgroup",
        "@org_golang_x_sys//unix",
    ],
)

go_test(
    name = "ociruntime_test",
    srcs = ["ociruntime_test.go"],
    data = [
        ":busybox",
        ":crun",
        "//enterprise/server/cmd/executor:tini",
        "//enterprise/server/remote_execution/runner/testworker",
        "@busybox",
    ],
    exec_properties = {
        "test.workload-isolation-type": "firecracker",
        "test.container-image": "docker://gcr.io/flame-public/net-tools@sha256:e904e2149194b94f4504fdfb3c7b2b71afa130708feb482fe9a9d557453fa8fd",
        "test.EstimatedComputeUnits": "2",
        "test.EstimatedFreeDiskBytes": "10GB",
    },
    # NOTE: if testing locally, use --test_sharding_strategy=disabled to work
    # around the networking package not supporting cross-process locks.
    shard_count = 4,
    tags = [
        "docker",
        "no-sandbox",
    ],
    target_compatible_with = [
        "@platforms//os:linux",
        # TODO: GitHub Actions arm runners are running into cgroup-related issues;
        # fix and re-enable on arm64.
        "@platforms//cpu:x86_64",
    ],
    x_defs = {
        "crunRlocationpath": "$(rlocationpath :crun)",
        "busyboxRlocationpath": "$(rlocationpath :busybox)",
        "ociBusyboxRlocationpath": "$(rlocationpath @busybox)",
        "testworkerRlocationpath": "$(rlocationpath //enterprise/server/remote_execution/runner/testworker)",
        "tiniRlocationpath": "$(rlocationpath //enterprise/server/cmd/executor:tini)",
    },
    deps = [
        ":ociruntime",
        "//enterprise/server/remote_execution/container",
        "//enterprise/server/remote_execution/persistentworker",
        "//enterprise/server/remote_execution/platform",
        "//enterprise/server/remote_execution/workspace",
        "//enterprise/server/util/cpuset",
        "//enterprise/server/util/oci",
        "//proto:remote_execution_go_proto",
        "//proto:scheduler_go_proto",
        "//proto:worker_go_proto",
        "//server/interfaces",
        "//server/real_environment",
        "//server/testutil/testenv",
        "//server/testutil/testfs",
        "//server/testutil/testnetworking",
        "//server/testutil/testregistry",
        "//server/testutil/testshell",
        "//server/testutil/testtar",
        "//server/util/disk",
        "//server/util/networking",
        "//server/util/proto",
        "//server/util/status",
        "//server/util/testing/flags",
        "//server/util/uuid",
        "@com_github_google_go_containerregistry//pkg/crane",
        "@com_github_google_go_containerregistry//pkg/v1:pkg",
        "@com_github_google_go_containerregistry//pkg/v1/mutate",
        "@com_github_stretchr_testify//assert",
        "@com_github_stretchr_testify//require",
        "@io_bazel_rules_go//go/runfiles",
    ],
)

alias(
    name = "crun",
    actual = select({
        "@platforms//cpu:x86_64": "@com_github_containers_crun_crun-linux-amd64//file:crun",
        "@platforms//cpu:aarch64": "@com_github_containers_crun_crun-linux-arm64//file:crun",
    }),
)

alias(
    name = "busybox",
    actual = select({
        "@platforms//cpu:x86_64": "@net_busybox_busybox-linux-amd64//file:busybox",
        "@platforms//cpu:aarch64": "@net_busybox_busybox-linux-arm64//file:busybox",
    }),
)
