package collector

import (
	"runtime/debug"
	"strconv"
	"sync"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/zxzixuanwang/aliyun-exporter-go/cfg"
	"github.com/zxzixuanwang/aliyun-exporter-go/pkg/aliyun/networkInOut"
)

var InternetOutRateByConnectionRegion = "InternetOutRateByConnectionRegion"

func init() {
	registerCollector(changeInternet(InternetOutRateByConnectionRegion, false), defaultEnabled, NewInternetOutRateByConnectionRegionCollector)
}

type internetOutRateByConnectionRegionCollectorCollector struct {
	entries *prometheus.Desc
	log     log.Logger
}

func NewInternetOutRateByConnectionRegionCollector(logger log.Logger) (Collector, error) {
	label := snakeString(InternetOutRateByConnectionRegion)
	level.Info(logger).Log("label", label)
	return &internetOutRateByConnectionRegionCollectorCollector{
		entries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, label, "value"),
			" region monitor with the amount of in and out ",
			[]string{"network"}, nil,
		),
		log: logger,
	}, nil
}

func (c *internetOutRateByConnectionRegionCollectorCollector) Update(ch chan<- prometheus.Metric) error {
	defer func() {
		err := recover()
		if err != nil {
			level.Error(c.log).Log("触发recover", string(debug.Stack()), "err:", err)
		}
	}()
	nanms, err := networkInOut.NewAliyunNetworkMonitorService(
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
	wg.Add(len(cfg.ConfigCollection.RegionMonitor.List))
	for _, v := range cfg.ConfigCollection.RegionMonitor.List {
		in := networkInOut.DefaultInput
		in.Metrics = InternetOutRateByConnectionRegion
		in.LocalRegionId = v.LocalRegionId
		in.Namespace = v.Namespace
		in.GeographicSpanId = v.GeographicSpanId
		in.OppositeRegionId = v.OppositeRegionId
		in.CenId = v.CentId
		go func() {
			defer wg.Done()
			in := in
			valueString, err := nanms.GetInternetOutRateByConnectionRegion(&in)
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
			label := in.LocalRegionId + "_" + in.OppositeRegionId + "_" + (in.Metrics)
			ch <- prometheus.MustNewConstMetric(
				c.entries, prometheus.GaugeValue, value, label)
			level.Info(c.log).Log(label, value)
		}()
	}
	wg.Wait()
	return nil
}
