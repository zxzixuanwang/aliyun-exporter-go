package slbInOut

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	cms "github.com/aliyun/alibaba-cloud-sdk-go/services/cms"
)

type aliyun struct {
	Client *cms.Client
	Log    log.Logger
}

type aliyuntrafficNew interface {
	GetAcsSlbDashboard(in *Input) (string, error)
}

func NewTraffic(region string, accessToken string, secret string, log log.Logger) (aliyuntrafficNew, error) {
	client, err := cms.NewClientWithAccessKey(region, accessToken, secret)
	if err != nil {
		level.Error(log).Log("创建服务失败", err)
		return nil, err
	}
	return &aliyun{
		Client: client,
		Log:    log,
	}, nil
}

var DefaultInput = Input{
	TimePlus: 5,
	Length:   5,
	Period:   60,
}

type Input struct {
	TimePlus   int8
	Length     int8
	Period     int
	Metrics    string
	InstanceId string
	Port       string
	Protocol   string
	Namespace  string
}

type InOutPut struct {
	Res []Res `json:"res"`
}
type Res struct {
	Timestamp int64   `json:"timestamp"`
	Value     float64 `json:"value"`
}
type SLBOut struct {
	Res []ResPercent `json:"res,omitempty"`
}
type ResPercent struct {
	Timestamp int64   `json:"timestamp,omitempty"`
	Average   float64 `json:"Average,omitempty"`
	Maximum   float64 `json:"Maximum,omitempty"`
	Minimum   float64 `json:"Minimum,omitempty"`
}

func (a *aliyun) GetAcsSlbDashboard(in *Input) (string, error) {
	defer func() {
		err := recover()
		if err != nil {
			level.Error(a.Log).Log("触发recover", string(debug.Stack()), "err:", err)
		}
	}()
	level.Debug(a.Log).Log("获取ＳＬＢ信息")
	endTime := time.Now()
	startTime := endTime.Add(time.Duration(-1*int(in.TimePlus)) * time.Minute)
	request := cms.CreateDescribeMetricDataRequest()
	request.Scheme = "https"
	request.MetricName = in.Metrics
	request.Namespace = in.Namespace
	request.StartTime = startTime.String()
	request.EndTime = endTime.String()
	request.Length = strconv.Itoa(int(in.Length))
	request.Period = strconv.Itoa(in.Period)

	temp := make(map[string]string, 3)
	temp["instanceId"] = in.InstanceId
	temp["port"] = in.Port
	temp["protocol"] = in.Protocol
	tempByte, err := json.Marshal(temp)
	if err != nil {
		level.Error(a.Log).Log("解析json 失败，指标", in.Metrics, "err", err)
		return "", err
	}

	request.Dimensions = string(tempByte)
	response, err := a.Client.DescribeMetricData(request)
	if err != nil {
		level.Error(a.Log).Log("获取数据 失败，指标", in.Metrics, "err", err)
		return "", err
	}

	if response.Code != strconv.Itoa(http.StatusOK) {
		level.Error(a.Log).Log("获取数据 失败，指标", in.Metrics, "err", response.Message)
		return "", fmt.Errorf(response.Message)
	}

	level.Debug(a.Log).Log("response数据", response)
	response.Datapoints = `{"res":` + response.Datapoints + "}"
	de := json.NewDecoder(strings.NewReader(response.Datapoints))
	de.UseNumber()
	value := ""

	result := new(SLBOut)
	err = de.Decode(result)
	if err != nil {
		level.Error(a.Log).Log("获取数据解析 失败，指标", in.Metrics, "err", err)
		return "", err
	}
	if len(result.Res) != 0 {
		level.Debug(a.Log).Log("时间为", time.Unix(0, (result.Res[len(result.Res)-1].Timestamp)*1e6).String())
		value = strconv.FormatFloat(result.Res[len(result.Res)-1].Average, 'f', -1, 64)
	} else {
		value = "0"
	}

	level.Info(a.Log).Log("时间内获取数据" + in.Metrics)
	return value, nil
}
