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
	"os"
	"strings"
	"time"
)

var (
	port     = flag.String("p", "8080", "port")
	mongoUri = flag.String("m", "mongodb://localhost:27017", "mongo uri")
	mongoDB  = flag.String("d", "ezinstall", "mongo database")
	mongoCol = flag.String("c", "open_record", "mongo collection")
)

func init() {
	flag.Parse()
	{
		if v := os.Getenv("PORT"); v != "" {
			*port = v
		}
	}
	{
		if v := os.Getenv("MONGO_URI"); v != "" {
			*mongoUri = v
		}
	}
	{
		if v := os.Getenv("MONGO_DB"); v != "" {
			*mongoDB = v
		}
	}
	{
		if v := os.Getenv("MONGO_COL"); v != "" {
			*mongoCol = v
		}
	}
}

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
	driver.MustInitMongoClient(*mongoUri, *mongoDB, *mongoCol)
	engine := gin.Default()
	// 跨域
	engine.Use(func(context *gin.Context) {
		context.Header("Access-Control-Allow-Origin", "*")
		context.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		context.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
		if context.Request.Method == "OPTIONS" {
			context.AbortWithStatus(200)
			return
		}
		context.Next()
	})
	handlers := func(context *gin.Context) {
		type Req struct {
			ChannelId string `form:"channelId"`
			AppName   string `form:"appName"`
		}
		in := &Req{}
		_ = context.ShouldBindQuery(in)
		// 获取请求域名
		origin := context.GetHeader("Origin")
		// 提取域名, 把http://或者https://去掉
		origin = strings.TrimPrefix(origin, "http://")
		origin = strings.TrimPrefix(origin, "https://")
		// 把/后面的路径去掉
		origin = strings.Split(origin, "/")[0]
		ip, isChina, isChinaInland, region := getRegion(context)
		if in.ChannelId != "" && in.AppName != "" {
			log.Printf("channelId: %s, appName: %s, ip: %s, region: %+v, isChina: %v, isChinaInland: %v\n", in.ChannelId, in.AppName, ip, region, isChina, isChinaInland)
			m := &driver.OpenRecord{
				ChannelId:      in.ChannelId,
				AppName:        in.AppName,
				Ip:             ip,
				Origin:         origin,
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
	}
	engine.GET("/", handlers)
	engine.POST("/", handlers)
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
