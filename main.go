package main

import (
	_ "embed"
	"flag"
	"fmt"
	gin "github.com/gin-gonic/gin"
	"github.com/jmt-tg/ezinstall/driver"
	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"strings"
	"time"
)

var (
	port     = flag.String("p", "8080", "port")
	mongoUri = flag.String("m", "mongodb://localhost:27017", "mongo uri")
	mongoDB  = flag.String("d", "ezinstall", "mongo database")
	mongoCol = flag.String("c", "open_record", "mongo collection")
)

// 中国大陆的省份，不包括香港","澳门","台湾
var chinaOutlandProvince = []string{
	"香港",
	"澳门",
	"台湾",
}

func getRegion(context *gin.Context) (string, bool, bool, driver.Region) {
	var cliIp string
	if context.GetHeader("X-REAL-IP") != "" {
		cliIp = context.GetHeader("X-REAL-IP")
	} else if context.GetHeader("X-FORWARDED-FOR") != "" {
		cliIp = context.GetHeader("X-FORWARDED-FOR")
	} else {
		cliIp = context.ClientIP()
	}
	// 去除端口
	cliIp = strings.Split(cliIp, ":")[0]
	region := ip2Region(cliIp)
	// 中国大陆的省份，不包括香港","澳门","台湾
	isChina := false
	isChainInland := true
	if region.Country == "中国" {
		isChina = true
		for _, province := range chinaOutlandProvince {
			if province == region.Province {
				isChainInland = false
				break
			}
		}
	} else {
		isChainInland = false
	}
	return cliIp, isChina, isChainInland, region
}

func main() {
	flag.Parse()
	driver.MustInitMongoClient(*mongoUri, *mongoDB, *mongoCol)
	engine := gin.Default()
	engine.GET("/", func(context *gin.Context) {
		type Req struct {
			ChannelId string `json:"channelId"`
			AppName   string `json:"appName"`
		}
		in := &Req{}
		_ = context.ShouldBind(in)
		ip, isChina, isChinaInland, region := getRegion(context)
		if in.ChannelId != "" {
			m := &driver.OpenRecord{
				ChannelId:      in.ChannelId,
				AppName:        in.AppName,
				Ip:             ip,
				Region:         region,
				IsCountryChina: isChina,
				IsChinaInland:  isChinaInland,
				CreatedAt:      primitive.Timestamp{T: uint32(time.Now().Unix())},
			}
			go m.Insert()
		}
		context.JSON(200, gin.H{
			"code":          0,
			"msg":           "success",
			"data":          region,
			"isChinaCounty": isChina,
			"isChinaInland": isChinaInland,
			"isChina":       isChinaInland,
		})
	})
	fmt.Printf("listen on http://127.0.0.1:%s\n", *port)
	engine.Run(":" + *port)
}

//go:embed ip2region.xdb
var Ip2regionXdb []byte

var (
	_searcher *xdb.Searcher
)

func init() {
	var err error
	_searcher, err = xdb.NewWithBuffer(Ip2regionXdb)
	if err != nil {
		log.Fatalf("open xdb file error: %v", err)
	}
	ip2Region("114.114.114.114")
}

func ip2Region(ip string) driver.Region {
	if ip == "" {
		return driver.Region{}
	}
	split := strings.Split(ip, ",")
	for _, s := range split {
		str, _ := _searcher.SearchByStr(s)
		if str != "" {
			regionSplit := strings.Split(str, "|")
			if len(regionSplit) == 5 {
				return driver.Region{
					Country:  regionSplit[0],
					District: regionSplit[1],
					Province: regionSplit[2],
					City:     regionSplit[3],
					ISP:      regionSplit[4],
				}
			}
		}
	}
	return driver.Region{}
}
