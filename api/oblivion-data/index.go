package index

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

/*
func main() {
	fmt.Println("hello world!!!!!")
	TestTransaction("test233", "192.168.7.7")
}
*/
func Handler(w http.ResponseWriter, r *http.Request) {

	// veiws := ModinfoFirebase("test", "test")
	veiws := GetSearch(r)

	w.Header().Set("content-type", "application/json;charset=UTF-8")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	fmt.Fprintf(w, veiws)
}

/*获取URL参数并写入MongoDB数据库,返回JSON字符串*/
func GetSearch(request *http.Request) string {
	defer func() { //异常处理
		if err := recover(); err != nil {
			fmt.Println("ERROR:", "参数不存在!")
		}
	}()

	search := request.URL.Query()
	apikey := search["a"][0]
	if apikey != "db1d9099a36841a746f30b44f9b8a8f21a9b9fd4" {
		fmt.Println("ERROR:", "搞事情!?")
		return "ERROR"
	}
	ip := search["ip"][0]
	path := search["path"][0]

	// return TestTransaction(path, ip)
	return PostToMongoDB(path, ip)
}

func PostToMongoDB(page string, ip string) string {
	//建立连接
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

	collpv := client.Database("modinfo").Collection("pv")
	colluv := client.Database("modinfo").Collection("ip")

	reJsonPV, err := IncreasePV(nil, collpv, page)
	reJsonUV, err := IncreaseUV(nil, colluv, ip)

	result := fmt.Sprintf(`{%v,%v}`, reJsonPV, reJsonUV)
	fmt.Println(result)

	result = fmt.Sprintf(`{%v}`, reJsonPV)
	return result
}

/*事务*/
func TestTransaction(page string, ip string) string {
	ctx := context.Background()
	uri := GetURI("modinfoapi")
	clientOpts := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		panic(err)
	}
	defer func() { _ = client.Disconnect(ctx) }()
	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(2*time.Second))
	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
	collpv := client.Database("modinfo").Collection("pv", wcMajorityCollectionOpts)
	colluv := client.Database("modinfo").Collection("ip", wcMajorityCollectionOpts)

	// Step 2: Start a session and run the callback using WithTransaction.
	session, err := client.StartSession()
	defer session.EndSession(ctx)
	//开始事务
	result, err := session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		// var err error
		reJsonPV, err := IncreasePV(sessCtx, collpv, page)
		reJsonUV, err := IncreaseUV(sessCtx, colluv, ip)
		return fmt.Sprintf(`{"pv":%v,"uv":%v}`, reJsonPV, reJsonUV), err
	})
	fmt.Printf("result: %v\n", result)
	return result.(string)
}

func IncreasePV(sessCtx mongo.SessionContext, coll *mongo.Collection, page string) (interface{}, error) {
	/*浏览量自增*/
	// Important: You must pass sessCtx as the Context parameter to the operations for them to be executed in the
	// transaction.
	ctx := context.TODO()
	var err error
	var result bson.M  //页面浏览量
	var totalPV bson.M //所有页面浏览量
	coll.FindOne(ctx, bson.M{"page": page}).Decode(&result)
	coll.FindOne(ctx, bson.M{"page": "total_page_views"}).Decode(&totalPV)

	var views int32 //页面浏览量
	if result["views"] == nil {
		//文档不存在时,添加文档
		coll.InsertOne(ctx, bson.M{"page": page, "views": 0})
		coll.UpdateOne(ctx, bson.M{"page": page}, bson.M{"$inc": bson.M{"views": 1}})
		views = 1
	} else {
		//浏览量自增1
		views = result["views"].(int32) + 1 //interface转int32
		var returnMessage *mongo.UpdateResult
		returnMessage, err = coll.UpdateOne(ctx, bson.M{"page": page}, bson.M{"$inc": bson.M{"views": 1}})
		fmt.Println("pv集合修改了:", returnMessage.ModifiedCount, "个文档")
	}
	//所有页面浏览量自增1
	totalViews := totalPV["views"].(int32) + 1
	coll.UpdateOne(ctx, bson.M{"page": "total_page_views"}, bson.M{"$inc": bson.M{"views": 1}})
	return fmt.Sprintf(`"pv":%d,"totalpv":%d`, views, totalViews), err
}

func IncreaseUV(sessCtx mongo.SessionContext, coll *mongo.Collection, ip string) (interface{}, error) {
	/*访客数自增*/
	ctx := context.TODO()
	var err error
	var result bson.M         //查询结果
	var uniqueVisitors bson.M //UV
	coll.FindOne(ctx, bson.M{"ip": ip}).Decode(&result)
	coll.FindOne(ctx, bson.M{"ip": "unique_visitors"}).Decode(&uniqueVisitors)

	uv := uniqueVisitors["views"].(int32)
	if result["views"] == nil {
		coll.InsertOne(ctx, bson.M{"ip": ip, "views": 0})
		coll.UpdateOne(ctx, bson.M{"ip": ip}, bson.M{"$inc": bson.M{"views": 1}})
		//访客数自增1
		coll.UpdateOne(ctx, bson.M{"ip": "unique_visitors"}, bson.M{"$inc": bson.M{"views": 1}})
		uv++
	} else {
		//ip浏览量自增1
		// views := result["views"].(int32) + 1 //interface转int32
		var returnMessage *mongo.UpdateResult
		returnMessage, err = coll.UpdateOne(ctx, bson.M{"ip": ip}, bson.M{"$inc": bson.M{"views": 1}})
		fmt.Println("ip集合修改了:", returnMessage.ModifiedCount, "个文档")
	}

	return fmt.Sprintf(`"uv":%d`, uv), err
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
