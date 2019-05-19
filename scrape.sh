#!/usr/bin/env bash

# wget command is taken from
# https://github.com/killinux/jslinux-bellard

exec 3<>/dev/tcp/seashells.io/1337
wget -r -p -np -k $(cut -d ' ' -f 3 <(head -n1 <(cat <&3)))

