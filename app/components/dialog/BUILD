load("//rules/typescript:index.bzl", "ts_library")

package(default_visibility = ["//visibility:public"])

exports_files(glob(["*.css"]))

ts_library(
    name = "dialog",
    srcs = ["dialog.tsx"],
    deps = [
        "//:node_modules/@types/react",
        "//:node_modules/react",
        "//:node_modules/tslib",
    ],
)

ts_library(
    name = "simple_modal_dialog",
    srcs = ["simple_modal_dialog.tsx"],
    deps = [
        "//:node_modules/@types/react",
        "//:node_modules/react",
        "//:node_modules/tslib",
        "//app/components/button",
        "//app/components/dialog",
        "//app/components/modal",
        "//app/components/spinner",
    ],
)
