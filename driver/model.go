package driver

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"log"
	"time"
)

type Region struct {
	Country  string `json:"country" bson:"country"`
	District string `json:"district" bson:"district"`
	Province string `json:"province" bson:"province"`
	City     string `json:"city" bson:"city"`
	ISP      string `json:"isp" bson:"isp"`
}

type OpenRecord struct {
	Id        primitive.ObjectID `bson:"_id,omitempty"`
	ChannelId string             `bson:"channel_id"`
	AppName   string             `bson:"app_name"`
	Ip        string             `bson:"ip"`
	Region    Region             `bson:"region"`
	// 域名
	Origin         string              `bson:"origin"`
	IsCountryChina bool                `bson:"is_country_china"`
	IsChinaInland  bool                `bson:"is_china_inland"`
	CreatedAt      primitive.Timestamp `bson:"created_at"`
}

func (m *OpenRecord) Insert() error {
	// 24小时内，ip只会记录一次
	// 1. 判断ip24小时内是否已经记录过
	foundCount, err := MongoCollection.CountDocuments(context.Background(), bson.M{
		"app_name": m.AppName,
		"ip":       m.Ip,
		"created_at": bson.M{
			"$gte": primitive.Timestamp{T: uint32(time.Now().Unix() - 24*60*60)},
		},
	})
	if err != nil {
		return err
	}
	if foundCount > 0 {
		log.Printf("ip: %s, 24小时内已经记录过", m.Ip)
		return nil
	}
	// 2. 记录ip
	_, err = MongoCollection.InsertOne(context.Background(), m)
	if err != nil {
		log.Printf("insert open record error: %v", err)
		return err
	}
	return nil
}
