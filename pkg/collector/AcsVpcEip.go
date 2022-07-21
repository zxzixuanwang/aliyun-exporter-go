package collector

import (
	"fmt"
	"runtime/debug"
	"strconv"
	"sync"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zxzixuanwang/aliyun-exporter-go/cfg"
	acsvpceip "github.com/zxzixuanwang/aliyun-exporter-go/pkg/aliyun/acsvpceip"
)

var AcsVpsEip = "AcsVpsEip"

func init() {
	registerCollector(snakeString(AcsVpsEip), defaultEnabled, NewAcsVpsEipCollector)
}

type acsVpsEip struct {
	entries *prometheus.Desc
	log     log.Logger
}

func NewAcsVpsEipCollector(logger log.Logger) (Collector, error) {
	label := snakeString(AcsVpsEip)
	level.Info(logger).Log("label", label)
	return &acsVpsEip{
		entries: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, label, "value"),
			" acs vpc eip average value",
			[]string{"eip"}, nil,
		),
		log: logger,
	}, nil
}

var metricsName = []string{"net.rx", "net.tx", "net.rxPkgs", "net.txPkgs", "net_rx.rate", "net_rxPkgs.rate", "net_tx.rate", "net_txPkgs.rate", "out_ratelimit_drop_speed", "net_in.rate_percentage", "net_out.rate_percentage"}

func (c *acsVpsEip) Update(ch chan<- prometheus.Metric) error {
	defer func() {
		err := recover()
		if err != nil {
			level.Error(c.log).Log("触发recover", string(debug.Stack()), "err:", err)
		}
	}()
	nanms, err := acsvpceip.NewAliyunNetworkMonitorService(
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
	wg.Add(len(metricsName))
	for _, v := range metricsName {
		in := acsvpceip.DefaultInput
		in.Metrics = v
		go func() {
			defer wg.Done()
			in := in

			getMapValue, err := nanms.GetAcsEipBy(&in)
			if err != nil {
				level.Error(c.log).Log("获取指标错误", err)
				return
			}
			level.Debug(c.log).Log("指标是:>>>>>>>>", in.Metrics)
			label := eipMetricsToMetrics(in.Metrics)

			entries := prometheus.NewDesc(
				prometheus.BuildFQName(namespace, label, "value"),
				fmt.Sprintf(" acs vpc eip %s", label),
				[]string{"region", "eip", "eip_instance", "public_ip"}, nil,
			)

			for k, v := range getMapValue {
				info := cfg.ConfigCollection.EipMonitor.MapList[k]
				newFloatValue, err := strconv.ParseFloat(v.Value, 64)
				if err != nil {
					level.Error(c.log).Log("转换指标错误:", err.Error())
					continue
				}

				ch <- prometheus.MustNewConstMetric(
					entries, prometheus.GaugeValue, newFloatValue, info.LocalRegionId, label, k, info.PublicAddress)

				level.Info(c.log).Log("instance", k, "public ip", info.PublicAddress, label, newFloatValue)
			}
		}()
		time.Sleep(time.Millisecond * time.Duration(cfg.ConfigCollection.Eip.Sleep))
	}
	wg.Wait()
	return nil
}
