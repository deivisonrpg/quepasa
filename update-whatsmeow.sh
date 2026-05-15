#!/usr/bin/env bash

set -u

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

if command -v tput >/dev/null 2>&1 && [[ -t 1 ]]; then
  CYAN="$(tput setaf 6)"
  YELLOW="$(tput setaf 3)"
  GREEN="$(tput setaf 2)"
  RED="$(tput setaf 1)"
  GRAY="$(tput setaf 7)"
  RESET="$(tput sgr0)"
else
  CYAN=""
  YELLOW=""
  GREEN=""
  RED=""
  GRAY=""
  RESET=""
fi

log_cyan() { printf '%s%s%s\n' "$CYAN" "$*" "$RESET"; }
log_yellow() { printf '%s%s%s\n' "$YELLOW" "$*" "$RESET"; }
log_green() { printf '%s%s%s\n' "$GREEN" "$*" "$RESET"; }
log_red() { printf '%s%s%s\n' "$RED" "$*" "$RESET"; }
log_gray() { printf '%s%s%s\n' "$GRAY" "$*" "$RESET"; }

update_qp_version() {
  local defaults_file
  local year
  local date
  local time
  local new_version
  local tmp_file

  defaults_file="$SCRIPT_DIR/src/models/qp_defaults.go"

  if [[ -f "$defaults_file" ]]; then
    log_cyan 'Updating QpVersion in qp_defaults.go...'

    year="$(date +%y)"
    date="$(date +%m%d)"
    time="$(date +%H%M)"
    new_version="3.$year.$date.$time"

    tmp_file="${defaults_file}.tmp"

    if sed -E "s/const QpVersion = \"[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+\"/const QpVersion = \"$new_version\"/" \
      "$defaults_file" > "$tmp_file"; then
      mv "$tmp_file" "$defaults_file"
      log_green "  Updated QpVersion to: $new_version"
      printf '\n'
    else
      rm -f "$tmp_file"
      log_red '  Failed to update QpVersion'
      printf '\n'
    fi
  else
    log_yellow '  Warning: qp_defaults.go not found'
    printf '\n'
  fi
}

log_cyan '=== Updating go.mau.fi/whatsmeow to @latest in all modules ==='
printf '\n'

update_qp_version

modules=(
  'src'
  'src/api'
  'src/docs'
  'src/environment'
  'src/form'
  'src/library'
  'src/media'
  'src/metrics'
  'src/models'
  'src/rabbitmq'
  'src/sipproxy'
  'src/webserver'
  'src/whatsapp'
  'src/whatsmeow'
)

root_dir="$SCRIPT_DIR"
success_count=0
fail_count=0

for module in "${modules[@]}"; do
  module_path="$root_dir/$module"

  if [[ -f "$module_path/go.mod" ]]; then
    log_yellow "Processing module: $module"
    pushd "$module_path" >/dev/null || continue

    if go get go.mau.fi/whatsmeow@latest; then
      log_green "  Successfully updated in $module"
      go mod tidy
      success_count=$((success_count + 1))
    else
      log_red "  Failed to update in $module"
      fail_count=$((fail_count + 1))
    fi

    popd >/dev/null || true
    printf '\n'
  else
    log_gray "Skipping $module (no go.mod found)"
    printf '\n'
  fi
done

log_cyan '=== Update Summary ==='
log_green "Successfully updated: $success_count modules"
log_red "Failed: $fail_count modules"
printf '\n'

log_cyan '=== Building the project ==='
pushd "$root_dir/src" >/dev/null || exit 1

if go build -o '../.dist/quepasa'; then
  log_green '  Build successful!'
else
  log_red '  Build failed!'
fi

popd >/dev/null || true
printf '\n'
log_cyan 'Done!'