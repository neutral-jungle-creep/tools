#!/bin/sh
set -e

COVERAGE_FILE="${1:-.coverage/unit.out}"
THRESHOLD="${2:-50}"
EXCLUDE_PATTERN="${3:-}"

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "ERROR: coverage file not found: $COVERAGE_FILE"
    exit 1
fi

TARGET_FILE="$COVERAGE_FILE"
if [ -n "$EXCLUDE_PATTERN" ]; then
    TARGET_FILE="${COVERAGE_FILE}.filtered"
    awk -v pat="$EXCLUDE_PATTERN" 'NR==1 || $0 !~ pat' "$COVERAGE_FILE" > "$TARGET_FILE"
fi

COVER_OUTPUT=$(go tool cover -func="$TARGET_FILE") || exit 1
PCT=$(printf '%s\n' "$COVER_OUTPUT" | awk '$1 == "total:" {gsub(/%/, "", $NF); print $NF}')

if [ -z "$PCT" ]; then
    echo "ERROR: total coverage not found"
    exit 1
fi

printf "Coverage: %s%% (threshold: %s%%)\n" "$PCT" "$THRESHOLD"
awk -v pct="$PCT" -v threshold="$THRESHOLD" 'BEGIN {
    if (pct + 0 < threshold + 0) {
        printf "FAIL: coverage %.1f%% is below threshold %s%%\n", pct + 0, threshold
        exit 1
    }
    printf "PASS: coverage %.1f%% >= threshold %s%%\n", pct + 0, threshold
}'
