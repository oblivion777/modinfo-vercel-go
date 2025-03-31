package index

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
	// defer func() { //异常处理
	// 	if err := recover(); err != nil {
	// 		fmt.Println("ERROR:", "参数不存在!")
	// 	}
	// }()

	search := request.URL.Query()
	ciphertext, _ := hex.DecodeString(search["c"][0])
	iv, _ := hex.DecodeString(search["v"][0])

	ip, path := GetVisitsData(ciphertext, iv)

	// ip := search["ip"][0]
	// path := search["path"][0]

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

	collpv := client.Database("commlist").Collection("pv")
	colluv := client.Database("commlist").Collection("ip")

	reJsonPV, err := IncreasePV(nil, collpv, page)
	reJsonUV, err := IncreaseUV(nil, colluv, ip)

	result := fmt.Sprintf(`{%v,%v}`, reJsonPV, reJsonUV)
	fmt.Println(result)

	result = fmt.Sprintf(`{%v}`, reJsonPV)
	return result
}

/*事务*/
// func TestTransaction(page string, ip string) string {
// 	ctx := context.Background()
// 	uri := GetURI("modinfoapi")
// 	clientOpts := options.Client().ApplyURI(uri)
// 	client, err := mongo.Connect(ctx, clientOpts)
// 	if err != nil {
// 		panic(err)
// 	}
// 	defer func() { _ = client.Disconnect(ctx) }()
// 	wcMajority := writeconcern.New(writeconcern.WMajority(), writeconcern.WTimeout(2*time.Second))
// 	wcMajorityCollectionOpts := options.Collection().SetWriteConcern(wcMajority)
// 	collpv := client.Database("modinfo").Collection("pv", wcMajorityCollectionOpts)
// 	colluv := client.Database("modinfo").Collection("ip", wcMajorityCollectionOpts)

// 	// Step 2: Start a session and run the callback using WithTransaction.
// 	session, err := client.StartSession()
// 	defer session.EndSession(ctx)
// 	//开始事务
// 	result, err := session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
// 		// var err error
// 		reJsonPV, err := IncreasePV(sessCtx, collpv, page)
// 		reJsonUV, err := IncreaseUV(sessCtx, colluv, ip)
// 		return fmt.Sprintf(`{"pv":%v,"uv":%v}`, reJsonPV, reJsonUV), err
// 	})
// 	fmt.Printf("result: %v\n", result)
// 	return result.(string)
// }

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

/*
	AES加密URL参数
*/
//解密函数
func CBCDecrypter(encrypter []byte, key []byte, iv []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		fmt.Println(err)
	}
	iv = IVPadding(iv)
	blockMode := cipher.NewCBCDecrypter(block, iv[:block.BlockSize()])
	result := make([]byte, len(encrypter))
	blockMode.CryptBlocks(result, encrypter)
	// 去除填充
	result = UnPKCS7Padding(result)
	return result
}

func UnPKCS7Padding(text []byte) []byte {
	// 取出填充的数据 以此来获得填充数据长度
	unPadding := int(text[len(text)-1])
	return text[:(len(text) - unPadding)]
}

// iv填充0
func IVPadding(sourceIV []byte) []byte {
	iv := [64]byte{}
	for i := 0; i < len(sourceIV); i++ {
		iv[i] = sourceIV[i]
	}
	return iv[:]
}

// return ip,path
func GetVisitsData(ciphertext []byte, iv []byte) (string, string) {
	type VisitsData struct {
		// A    string `json:"a"`
		Ip   string `json:"ip"`
		Path string `json:"path"`
	}
	AES_KEY, _ := hex.DecodeString(os.Getenv("AES_KEY"))
	data := CBCDecrypter(ciphertext, AES_KEY, iv)
	var visitsData VisitsData
	json.Unmarshal(data, &visitsData)
	return visitsData.Ip, visitsData.Path
}
