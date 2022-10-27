package exportdata

import (
	"runtime/debug"
	"time"

	cms_export20211101 "github.com/alibabacloud-go/cms-export-20211101/client"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/zxzixuanwang/aliyun-exporter-go/tools"
)

func BatchGet[T any](client *cms_export20211101.Client,
	in *Input, l log.Logger, f func(in *cms_export20211101.BatchGetResponse) map[string]*T) (map[string]*T, error) {
	defer func() {
		err := recover()
		if err != nil {
			level.Error(l).Log("触发recover", string(debug.Stack()), "err:", err)
		}
	}()
	level.Info(l).Log("获取eip,指标%s", in.Metrics)

	endTime := time.Now().Local()
	endTimeInt := endTime.UnixNano() / 1e6
	startTime := endTime.Add(time.Duration(-1*int(in.TimePlus)) * time.Minute)
	startTimeInt := startTime.UnixNano() / 1e6
	cursorRequest := &cms_export20211101.CursorRequest{
		EndTime:   &endTimeInt,
		StartTime: &startTimeInt,
		Namespace: &in.Namespace,
		Period:    tools.GetPoint(int32(in.Period)),
		Metric:    &in.Metrics,
	}

	o, err := client.Cursor(cursorRequest)
	if err != nil {
		level.Error(l).Log("获取 cursor 指标", in.Metrics, "err is", err)
		return nil, err
	}
	if o.Body.Data == nil {
		level.Error(l).Log("获取 cursor 指标", in.Metrics, "err is", o.Body.Message)
		return nil, err
	}
	c := o.Body.Data.Cursor
	get, err := client.BatchGet(&cms_export20211101.BatchGetRequest{
		Namespace: cursorRequest.Namespace,
		Metric:    cursorRequest.Metric,
		Cursor:    c,
		Length:    tools.GetPoint(int32(1000)),
	})
	if err != nil {
		level.Error(l).Log("获取 batch 指标", in.Metrics, "err is", err)
		return nil, err
	}

	if get.Body.Data == nil {
		level.Error(l).Log("获取 batch data 失败 指标", in.Metrics, "err is", o.Body.Message)
		return nil, err
	}
	return f(get), nil
}

type Input struct {
	Namespace  string
	Metrics    string
	TimePlus   int8
	Length     int8
	Period     int
	InstanceId string
}
