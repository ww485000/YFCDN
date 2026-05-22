#!/usr/bin/env bash
set -euo pipefail

REPO_URL="${1:-https://github.com/ww485000/YFCDN.git}"
BRANCH="${2:-${BRANCH:-main}}"
COMMIT_MSG="${COMMIT_MSG:-upgrade YFCDN to v0.1.3}"
FORCE="${FORCE:-0}"

if ! command -v git >/dev/null 2>&1; then
  echo "git is required" >&2
  exit 1
fi

git init >/dev/null
git checkout -B "$BRANCH" >/dev/null

if git remote get-url origin >/dev/null 2>&1; then
  git remote set-url origin "$REPO_URL"
else
  git remote add origin "$REPO_URL"
fi

git add .
if git diff --cached --quiet; then
  echo "No source changes to commit."
else
  git commit -m "$COMMIT_MSG"
fi

if [ "$FORCE" = "1" ]; then
  git push -u origin "$BRANCH" --force
else
  git push -u origin "$BRANCH"
fi
