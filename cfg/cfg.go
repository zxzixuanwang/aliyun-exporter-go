package cfg

import (
	"fmt"
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/pvtz"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/vpc"
	"github.com/robfig/cron"
	"github.com/spf13/viper"
)

var ConfigCollection ConfigMap
var DefaultConfig = ConfigMap{
	Services: Service{Env: "test", Port: ":8000", Name: "aliyun-exporter"},
	Log: LogConfig{
		Level: "debug",
	},
}

type Eip struct {
	Sleep    int
	EndPoint string
}

type RegionMonitor struct {
	List []RegionMonitorList
}
type RegionMonitorList struct {
	CentId           string
	GeographicSpanId string
	LocalRegionId    string
	OppositeRegionId string
	Namespace        string
}

type ConfigMap struct {
	Log           LogConfig
	Services      Service
	Aliyun        Aliyun
	RegionMonitor RegionMonitor
	SlbMonitor    SlbMonitor
	EipMonitor    EipMonitor
	Eip           Eip
}

type LogConfig struct {
	Level string
}

type Aliyun struct {
	AccessToken string
	SecretToken string
	Region      string
	SkipRegion  []string
}
type File struct {
	Enabled    bool
	Level      string
	Path       string
	Name       string
	MaxHistory int
	MaxSizeMb  int
}
type SlbMonitor struct {
	List []SlbMonitorList
}
type SlbMonitorList struct {
	Namespace  string
	InstanceId string
	Port       string
	Protocol   string
}
type Service struct {
	Env         string
	Port        string
	Name        string
	MetricsOpen bool
	MaxRequest  int
	MetricPath  string
}

type EipMonitor struct {
	List    []EipMonitorList
	MapList map[string]EipMonitorList
}
type EipMonitorList struct {
	Namespace     string
	InstanceId    string
	LocalRegionId string
	PublicAddress string
	//Name                 string
	//ReservationBandwidth string
}

func init() {
	viper.AddConfigPath("./config")
	viper.SetConfigName("aliyun-exporter")
	viper.SetConfigType("yaml")
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		panic("read config err:" + err.Error())
	}

	ConfigCollection = DefaultConfig
	err = viper.Unmarshal(&ConfigCollection)
	if err != nil {
		panic("get config error:" + err.Error())
	}

	regionList := getEipRegion()

	searchAllEip(true, regionList)

	go crontab(regionList)
}

func crontab(regionList []string) {
	c := cron.New()

	specEipList := "0 0 * * *"
	c.AddFunc(specEipList, func() {
		searchAllEip(false, regionList)
	})

	c.Start()
}

func getEipRegion() []string {

	client, err := pvtz.NewClientWithAccessKey(ConfigCollection.Aliyun.Region, ConfigCollection.Aliyun.AccessToken, ConfigCollection.Aliyun.SecretToken)
	if err != nil {
		panic("get eip region " + err.Error())
	}
	request := pvtz.CreateDescribeRegionsRequest()
	request.Scheme = "https"

	response, err := client.DescribeRegions(request)
	if err != nil {
		fmt.Print(err.Error())
		return nil
	}

	regionList := make([]string, 0, len(response.Regions.Region))
	for _, v := range response.Regions.Region {
		regionList = append(regionList, v.RegionId)
	}

	return regionList
}

func searchAllEip(first bool, list []string) {
	listEip := make([]EipMonitorList, 0, 200)

	for _, v := range list {
		fmt.Println("search ", v)
		listOne := searchEip(first, v)

		if len(listOne) > 0 {
			listEip = append(listEip, listOne...)
		}
	}

	ConfigCollection.EipMonitor.List = listEip
	tmpMapEip := make(map[string]EipMonitorList, len(listEip))
	for k, v := range ConfigCollection.EipMonitor.List {
		tmpMapEip[v.InstanceId] = v
		fmt.Printf("eip monitor第%d 配置\n", k+1)
		fmt.Println("eip 实例ID", v.InstanceId)
		fmt.Println("eip 可用区域", v.LocalRegionId)
	}
	ConfigCollection.EipMonitor.MapList = tmpMapEip
	fmt.Println("tmpMapEip: >>>>>", tmpMapEip)
}

func searchEip(first bool, region string) []EipMonitorList {
	for _, v := range ConfigCollection.Aliyun.SkipRegion {
		if region == v {
			return []EipMonitorList{}
		}
	}

	client, err := vpc.NewClientWithAccessKey(region, ConfigCollection.Aliyun.AccessToken, ConfigCollection.Aliyun.SecretToken)

	if err != nil {
		if first {
			panic("请求client " + err.Error())
		} else {
			fmt.Println("请求eip list 失败:", err)
			return nil
		}
	}

	size := 100
	page := 1
	tempTotal := 0
	total := 1
	listEip := make([]EipMonitorList, 0, 100)

	request := vpc.CreateDescribeEipAddressesRequest()
	request.Scheme = "https"
	request.PageSize = requests.NewInteger(size)

	for total > tempTotal {

		request.PageNumber = requests.NewInteger(page)
		response, err := client.DescribeEipAddresses(request)

		if err != nil {
			if first {
				panic("获取eip list" + err.Error())
			} else {
				fmt.Println("请求eip list 失败:", err)
				return nil
			}
		}

		total = response.TotalCount
		handleList(&listEip, response)
		tempTotal += size
		page++
	}
	return listEip
}

func handleList(listEip *[]EipMonitorList, response *vpc.DescribeEipAddressesResponse) {
	for _, v := range response.EipAddresses.EipAddress {
		if v.Status == "InUse" {
			listOne := EipMonitorList{
				Namespace:     "acs_vpc_eip",
				InstanceId:    v.AllocationId,
				PublicAddress: v.IpAddress,
			}

			if len(v.AvailableRegions.AvailableRegion) > 0 {
				if strings.Contains(v.AvailableRegions.AvailableRegion[0], "-") {
					region := strings.Split(v.AvailableRegions.AvailableRegion[0], "-")[1]
					listOne.LocalRegionId = region
				}
			} else {
				fmt.Printf("%s 没有可用区域", v.AllocationId)
				continue
			}

			*listEip = append(*listEip, listOne)
		}
	}
}
