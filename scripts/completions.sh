#!/bin/sh
# Generates shell completions for release archives and the Homebrew cask.
set -e
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
	go run ./cmd/fanout completion "$sh" >"completions/fanout.$sh"
done
