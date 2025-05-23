#!/bin/bash
set -euo pipefail

export RUNFILES_DIR=$(cd ../ && pwd)

# Use absolute paths as we cd to the WORKSPACE directory below.
GO_PATH="$(pwd)/go"
GOFMT_PATH="$(pwd)/gofmt"
GOIMPORTS_PATH="$(pwd)/tools/goimports/goimports.sh"
BB_PATH="$(pwd)/cli/cmd/bb/bb_/bb"
PRETTIER_PATH="$(pwd)/tools/prettier/prettier.sh"
CLANG_FORMAT_PATH="$(pwd)/tools/clang-format/clang-format"

# Make sure 'go' is in $PATH (gazelle depends on this).
# TODO: set up the env properly to point to the bazel-provisioned SDK root
export PATH="$PATH:$PWD"

cd "$BUILD_WORKSPACE_DIRECTORY"

function timestamp() {
  date '+%Y-%m-%d %H:%M:%S'
}

function info() {
  echo >&2 -e "\x1b[90m$(timestamp)\x1b[0m\x1b[96m INFO:\x1b[m" "$@" "\x1b[0m"
}

function error() {
  echo >&2 -e "\x1b[90m$(timestamp)\x1b[0m\x1b[91m ERROR:\x1b[m" "$@" "\x1b[0m"
}

LOGDIR=$(mktemp -d)
trap 'rm -r "$LOGDIR"' EXIT
LOGFILES=()

function run() {
  NAME="$1"
  info "Starting $NAME"
  shift
  LOGFILES+=("$NAME")
  {
    "$@" >"$LOGDIR/$NAME" 2>&1 || true
    info "Finished $NAME"
  } &
}

run BuildFiles \
  "$BB_PATH" fix --diff

run GoDeps \
  env GO_PATH="$GO_PATH" \
  tools/fix_go_deps.sh --diff

run GoFormat \
  "$GOFMT_PATH" -d .

run GoImports \
  "$GOIMPORTS_PATH" -d .

run ProtoFormat \
  env CLANG_FORMAT_PATH="$CLANG_FORMAT_PATH" \
  tools/clang-format/clang-format.sh --dry-run

run PrettierFormat \
  "$PRETTIER_PATH" --log-level=warn --check

wait

OK=1
for LOGFILE in "${LOGFILES[@]}"; do
  CONTENT=$(cat "$LOGDIR/$LOGFILE")
  if [[ "$CONTENT" ]]; then
    error "$LOGFILE:"
    echo "$CONTENT"
    OK=0
  fi
done

if ! ((OK)); then
  error "Some checks failed. To fix many common errors, run: ./buildfix.sh -a"
  exit 1
fi

info "All checks passed!"
