package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	fmt.Println("hello world!!!!!")
	PostToMongoDB()
}

func PostToMongoDB() {
	uri := GetURI("modinfoapi")
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := client.Disconnect(context.TODO()); err != nil {
			panic(err)
		}
	}()

	coll := client.Database("modinfo").Collection("pv")
	colluv := client.Database("modinfo").Collection("ip")
	page := "test"
	var result bson.M

	IncreasePV(nil, coll, "test")
	IncreaseUV(nil, colluv, "119.29.29.29")
	err = coll.FindOne(context.TODO(), bson.D{{"page", page}}).Decode(&result)
	jsonData, err := json.MarshalIndent(result, "", "    ")
	fmt.Printf("%s\n", jsonData)

}

func IncreasePV(sessCtx mongo.SessionContext, coll *mongo.Collection, page string) (interface{}, error) {
	//浏览量自增1
	// Important: You must pass sessCtx as the Context parameter to the operations for them to be executed in the
	// transaction.
	var err error
	var result bson.M  //页面浏览量
	var totalPV bson.M //所有页面浏览量
	coll.FindOne(context.TODO(), bson.D{{"page", page}}).Decode(&result)
	coll.FindOne(context.TODO(), bson.D{{"page", "total_page_views"}}).Decode(&totalPV)

	var views int32 //页面浏览量
	if result["views"] == nil {
		//文档不存在时,添加文档
		coll.InsertOne(context.TODO(), bson.D{{"page", page}, {"views", 1}})
		views = 1
	} else {
		//浏览量自增1
		views = result["views"].(int32) + 1 //interface转int32
		var returnMessage *mongo.UpdateResult
		returnMessage, err = coll.UpdateOne(context.TODO(), bson.D{{"page", page}}, bson.D{{"$set", bson.D{{"views", views}}}})
		fmt.Println(returnMessage.MatchedCount)
	}
	//所有页面浏览量自增1
	totalViews := totalPV["views"].(int32) + 1
	coll.UpdateOne(context.TODO(), bson.D{{"page", "total_page_views"}}, bson.D{{"$set", bson.D{{"views", totalViews}}}})
	return fmt.Sprintf(`{"pv":%d,"totalpv":%d}`, views, totalViews), err
}

func IncreaseUV(sessCtx mongo.SessionContext, coll *mongo.Collection, ip string) (interface{}, error) {
	//访客数+1
	var err error
	var result bson.M         //查询结果
	var uniqueVisitors bson.M //UV
	coll.FindOne(context.TODO(), bson.D{{"ip", ip}}).Decode(&result)
	coll.FindOne(context.TODO(), bson.D{{"ip", "unique_visitors"}}).Decode(&uniqueVisitors)

	uv := uniqueVisitors["views"].(int32)
	if result["views"] == nil {
		coll.InsertOne(context.TODO(), bson.D{{"ip", ip}, {"views", 1}})
		//总浏览量自增1
		views := uv + 1
		coll.UpdateOne(context.TODO(), bson.D{{"ip", "unique_visitors"}}, bson.D{{"$set", bson.D{{"views", views}}}})
	} else {
		//ip浏览量自增1
		views := result["views"].(int32) + 1 //interface转int32
		var returnMessage *mongo.UpdateResult
		returnMessage, err = coll.UpdateOne(context.TODO(), bson.D{{"ip", ip}}, bson.D{{"$set", bson.D{{"views", views}}}})
		fmt.Println(returnMessage.MatchedCount)
	}

	return fmt.Sprintf(`{"uv":%d}`, uv), err
}

func GetURI(user string) string {
	// user := "modinfoapi"
	var key string

	switch user {
	case "modinfoapi":
		key = os.Getenv("MODINFOAPI_OBLIVION_MONGODB")
	default:
		key = "ERROR!!!"
	}

	reURI := fmt.Sprintf(`mongodb+srv://%s:%s@reminiscence.lhull.mongodb.net/myFirstDatabase?retryWrites=true&w=majority`, user, key)

	return reURI
}
