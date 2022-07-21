#!/bin/sh
chmod +x aliyun-exporter
nohup ./aliyun-exporter > app.log 2>&1 &
echo $! > PID
