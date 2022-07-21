#!/bin/sh
local=`pwd`
rm -rf $local/package
mkdir -p $local/package/config
go build -o $local/package/aliyun-exporter

cp start.sh stop.sh $local/package
chmod +x $local/package/start.sh $local/package/stop.sh
cp config/*.yaml $local/package/config