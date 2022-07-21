package collector

import (
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zxzixuanwang/aliyun-exporter-go/cfg"
	"github.com/zxzixuanwang/aliyun-exporter-go/pkg/aliyun/slbInOut"
)

var InstanceTrafficTX = "InstanceTrafficTX"

func init() {
	registerCollector("Aliyun_TrafficTXNew", defaultEnabled, NewtrafficTXNew)
}

type trafficTXNew struct {
	entries *prometheus.Desc
	log     log.Logger
}

func NewtrafficTXNew(logger log.Logger) (Collector, error) {
	label := snakeString(InstanceTrafficTX)
	level.Info(logger).Log("label", label)
	return &trafficTXNew{
		entries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, label, "value"),
			" SLB in and out ",
			[]string{"slbInOut"}, nil,
		),
		log: logger,
	}, nil
}

func (c *trafficTXNew) Update(ch chan<- prometheus.Metric) error {
	defer func() {
		err := recover()
		if err != nil {
			level.Error(c.log).Log("触发recover", string(debug.Stack()), "err:", err)
		}
	}()
	nanms, err := slbInOut.NewTraffic(

		cfg.ConfigCollection.Aliyun.Region,
		cfg.ConfigCollection.Aliyun.AccessToken,
		cfg.ConfigCollection.Aliyun.SecretToken,
		c.log,
	)

	if err != nil {
		level.Error(c.log).Log("更新指标错误", err)
		return err
	}
	wg := sync.WaitGroup{}
	wg.Add(len(cfg.ConfigCollection.SlbMonitor.List))
	for _, v := range cfg.ConfigCollection.SlbMonitor.List {
		in := slbInOut.DefaultInput
		in.Metrics = InstanceTrafficTX
		in.Namespace = v.Namespace
		in.Port = v.Port
		in.Protocol = v.Protocol
		in.InstanceId = v.InstanceId
		go func() {
			defer wg.Done()
			in := in
			valueString, err := nanms.GetAcsSlbDashboard(&in)
			if err != nil {
				level.Error(c.log).Log("获取指标错误", err)
				return
			}
			value, err := strconv.ParseFloat(valueString, 64)
			if err != nil {
				level.Error(c.log).Log("转换指标错误", err, "value", valueString)
				return
			}
			level.Debug(c.log).Log("指标是:>>>>>>>>", in.Metrics)
			label := "SLB_" + in.Metrics
			ch <- prometheus.MustNewConstMetric(
				c.entries, prometheus.GaugeValue, value, label)
			level.Info(c.log).Log(label, value)
		}()
	}
	wg.Wait()
	return nil
}
