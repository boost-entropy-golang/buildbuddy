load("//rules/typescript:index.bzl", "ts_library")

package(default_visibility = ["//visibility:public"])

ts_library(
    name = "setup",
    srcs = ["setup.tsx"],
    deps = [
        "//:node_modules/@types/react",
        "//:node_modules/react",
        "//:node_modules/tslib",
        "//app/auth:user",
        "//app/capabilities",
        "//app/docs:setup_code",
    ],
)

ts_library(
    name = "setup_code",
    srcs = ["setup_code.tsx"],
    deps = [
        "//:node_modules/@types/react",
        "//:node_modules/react",
        "//:node_modules/tslib",
        "//app/auth:auth_service",
        "//app/capabilities",
        "//app/components/banner",
        "//app/components/button:link_button",
        "//app/components/select",
        "//app/components/spinner",
        "//app/errors:error_service",
        "//app/service:rpc_service",
        "//proto:api_key_ts_proto",
        "//proto:bazel_config_ts_proto",
    ],
)
