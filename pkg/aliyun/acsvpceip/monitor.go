package acsvpceip

import (
	"github.com/go-kit/log"
	"github.com/zxzixuanwang/aliyun-exporter-go/cfg"
	"github.com/zxzixuanwang/aliyun-exporter-go/pkg/exportdata"
	"github.com/zxzixuanwang/aliyun-exporter-go/tools"

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
		Endpoint:        tools.GetPoint(cfg.ConfigCollection.Eip.EndPoint),
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
	in.TimePlus = 2
	searchInput := &exportdata.Input{
		Namespace:  in.Namespace,
		Metrics:    in.Metrics,
		TimePlus:   in.TimePlus,
		Length:     in.Length,
		Period:     in.Period,
		InstanceId: in.InstanceId,
	}

	f := func(get *cms_export20211101.BatchGetResponse) map[string]*LabelValue {

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
		return tmpMap
	}

	tmpMap, err := exportdata.BatchGet(a.Client, searchInput, a.Log, f)
	if err != nil {
		return nil, err
	}

	for k, v := range tmpMap {
		level.Info(a.Log).Log("时间内获取指标", in.Metrics, "实例", k, "值", v.Value)
	}

	return tmpMap, nil

}
