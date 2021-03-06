#!/usr/bin/env bash

set -ex

GOROOT=/usr/local/go
GO_ARCHIVE_URL=https://storage.googleapis.com/golang/go1.3.3.linux-amd64.tar.gz
GO_ARCHIVE=$(basename $GO_ARCHIVE_URL)

echo "Downloading go..."
mkdir -p $(dirname $GOROOT)
wget -q $GO_ARCHIVE_URL
tar xf $GO_ARCHIVE -C $(dirname $GOROOT)
chmod -R a+w $GOROOT

export GOROOT
export GOPATH=$GOROOT
export PATH=$GOROOT/bin:$PATH
