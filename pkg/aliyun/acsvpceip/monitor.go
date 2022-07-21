package acsvpceip

import (
	"runtime/debug"

	"github.com/go-kit/log"
	"github.com/zxzixuanwang/aliyun-exporter-go/cfg"
	"github.com/zxzixuanwang/aliyun-exporter-go/tools"

	"time"

	cms_export20211101 "github.com/alibabacloud-go/cms-export-20211101/client"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	"github.com/go-kit/log/level"
)

type AllStatistics struct {
	Res []AllRes `json:"res,omitempty"`
}

type ValueRes struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"Value"`
}

type ValueStatistics struct {
	Res []ValueRes `json:"res,omitempty"`
}

type AllRes struct {
	Timestamp int64   `json:"timestamp,omitempty"`
	Average   float64 `json:"Average,omitempty"`
	Maximum   float64 `json:"Maximum,omitempty"`
	Minimum   float64 `json:"Minimum,omitempty"`
}

type aliyun struct {
	Client *cms_export20211101.Client
	Log    log.Logger
}

type Input struct {
	Namespace  string
	Metrics    string
	TimePlus   int8
	Length     int8
	Period     int
	InstanceId string
}
type aliyunNetworkAceEip interface {
	GetAcsEipBy(in *Input) (map[string]*LabelValue, error)
}

var DefaultInput = Input{
	TimePlus:  5,
	Length:    5,
	Period:    60,
	Namespace: "acs_vpc_eip",
}

func NewAliyunNetworkMonitorService(region string, accessToken string, secret string, log log.Logger) (aliyunNetworkAceEip, error) {
	clientExport, err := cms_export20211101.NewClient(&openapi.Config{
		AccessKeyId:     &accessToken,
		AccessKeySecret: &secret,
		Endpoint:        tools.String2point(cfg.ConfigCollection.Eip.EndPoint),
	})
	if err != nil {
		level.Error(log).Log("获取err is", err)
		return nil, err
	}
	return &aliyun{
		Client: clientExport,
		Log:    log,
	}, nil
}

type LabelValue struct {
	Time  int64
	Value string
}

func (a *aliyun) GetAcsEipBy(in *Input) (map[string]*LabelValue, error) {
	defer func() {
		err := recover()
		if err != nil {
			level.Error(a.Log).Log("触发recover", string(debug.Stack()), "err:", err)
		}
	}()
	in.TimePlus = 2
	level.Info(a.Log).Log("获取eip,指标%s", in.Metrics)

	endTime := time.Now().Local()
	endTimeInt := endTime.UnixNano() / 1e6
	startTime := endTime.Add(time.Duration(-1*int(in.TimePlus)) * time.Minute)
	startTimeInt := startTime.UnixNano() / 1e6
	cursorRequest := &cms_export20211101.CursorRequest{
		EndTime:   &endTimeInt,
		StartTime: &startTimeInt,
		Namespace: &in.Namespace,
		Period:    tools.Int322point(int32(in.Period)),
		Metric:    &in.Metrics,
	}

	o, err := a.Client.Cursor(cursorRequest)
	if err != nil {
		level.Error(a.Log).Log("获取 cursor 指标", in.Metrics, "err is", err)
		return nil, err
	}
	if o.Body.Data == nil {
		level.Error(a.Log).Log("获取 cursor 指标", in.Metrics, "err is", o.Body.Message)
		return nil, err
	}
	c := o.Body.Data.Cursor
	get, err := a.Client.BatchGet(&cms_export20211101.BatchGetRequest{
		Namespace: cursorRequest.Namespace,
		Metric:    cursorRequest.Metric,
		Cursor:    c,
		Length:    tools.Int322point(1000),
	})
	if err != nil {
		level.Error(a.Log).Log("获取 batch 指标", in.Metrics, "err is", err)
		return nil, err
	}

	if get.Body.Data == nil {
		level.Error(a.Log).Log("获取 batch data 失败 指标", in.Metrics, "err is", o.Body.Message)
		return nil, err
	}
	tmpMap := make(map[string]*LabelValue, 10)
	for _, v := range get.Body.Data.Records {
		if tmpMap[*v.LabelValues[1]] != nil {
			if *v.Timestamp > tmpMap[*v.LabelValues[1]].Time {
				if in.Metrics == "net_rx.rate" ||
					in.Metrics == "net_rxPkgs.rate" ||
					in.Metrics == "net_tx.rate" ||
					in.Metrics == "net_txPkgs.rate" {

					tmpMap[*v.LabelValues[1]] = &LabelValue{
						Time:  *v.Timestamp,
						Value: *v.MeasureValues[0],
					}
				} else {
					tmpMap[*v.LabelValues[1]] = &LabelValue{
						Time:  *v.Timestamp,
						Value: *v.MeasureValues[2],
					}
				}

			}
			continue
		}
		if in.Metrics == "net_rx.rate" ||
			in.Metrics == "net_rxPkgs.rate" ||
			in.Metrics == "net_tx.rate" ||
			in.Metrics == "net_txPkgs.rate" {

			tmpMap[*v.LabelValues[1]] = &LabelValue{
				Time:  *v.Timestamp,
				Value: *v.MeasureValues[0],
			}
		} else {
			tmpMap[*v.LabelValues[1]] = &LabelValue{
				Time:  *v.Timestamp,
				Value: *v.MeasureValues[2],
			}
		}

	}
	for k, v := range tmpMap {
		level.Info(a.Log).Log("时间内获取指标", in.Metrics, "实例", k, "值", v.Value)
	}

	return tmpMap, nil

}
