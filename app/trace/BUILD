load("//rules/typescript:index.bzl", "ts_jasmine_node_test", "ts_library")

package(default_visibility = ["//visibility:public"])

ts_library(
    name = "event_hovercard",
    srcs = ["event_hovercard.tsx"],
    deps = [
        "//:node_modules/@types/react",
        "//:node_modules/react",
        "//app/trace:trace_events",
        "//app/util:math",
    ],
)

ts_library(
    name = "trace_viewer",
    srcs = ["trace_viewer.tsx"],
    deps = [
        "//:node_modules/@types/react",
        "//:node_modules/lucide-react",
        "//:node_modules/react",
        "//:node_modules/tslib",
        "//app/components/filter_input",
        "//app/router",
        "//app/trace:constants",
        "//app/trace:event_hovercard",
        "//app/trace:trace_events",
        "//app/trace:trace_viewer_model",
        "//app/trace:trace_viewer_panel",
        "//app/util:animated_value",
        "//app/util:animation_loop",
        "//app/util:dom",
        "//app/util:math",
        "//app/util:platform",
    ],
)

ts_library(
    name = "trace_viewer_model",
    srcs = ["trace_viewer_model.ts"],
    deps = [
        "//:node_modules/tslib",
        "//app/trace:constants",
        "//app/trace:trace_events",
        "//app/util:color",
    ],
)

exports_files(glob(["*.css"]))

ts_library(
    name = "constants",
    srcs = ["constants.ts"],
)

ts_library(
    name = "trace_events",
    srcs = ["trace_events.ts"],
    deps = [
        "//:node_modules/tslib",
    ],
)

ts_jasmine_node_test(
    name = "trace_events_test",
    srcs = ["trace_events_test.ts"],
    deps = [":trace_events"],
)

ts_library(
    name = "trace_viewer_panel",
    srcs = ["trace_viewer_panel.ts"],
    deps = [
        "//:node_modules/tslib",
        "//app/trace:constants",
        "//app/trace:trace_events",
        "//app/trace:trace_viewer_model",
        "//app/util:canvas",
        "//app/util:dom",
        "//app/util:math",
    ],
)
