package index

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	firebase "firebase.google.com/go"
	"firebase.google.com/go/db"
	"google.golang.org/api/option"
)

func Handler(w http.ResponseWriter, r *http.Request) {

	// veiws := ModinfoFirebase("test", "test")
	// veiws := GetSearch(r)
	fmt.Fprintf(w, "veiws")
}

/*获取URL参数并写入Firebase Realtime数据库,返回JSON字符串*/
func GetSearch(request *http.Request) string {
	defer func() {
		if err := recover(); err != nil {
			log.Println("ERROR:", "参数不存在!")
		}
	}()

	search := request.URL.Query()
	apikey := search["a"][0]
	if apikey != "db1d9099a36841a746f30b44f9b8a8f21a9b9fd4" {
		log.Println("ERROR:", "搞事情!?")
		return "ERROR"
	}
	ip := search["ip"][0]
	path := search["path"][0]

	views := ModinfoFirebase(ip, path)
	return views
}

func ModinfoFirebase(ip string, path string) string {
	APIKEY := os.Getenv("GOOGEL_REALTIME_DATABASE_KEY")

	var nilMap map[string]interface{} //缩小 Admin SDK 的权限范围，以将其用作未经身份验证的客户端
	ctx := context.Background()
	conf := &firebase.Config{
		DatabaseURL:  "https://modinfobaas-default-rtdb.firebaseio.com/",
		AuthOverride: &nilMap, //缩小 Admin SDK 的权限范围，以将其用作未经身份验证的客户端
	}
	// Fetch the service account key JSON file contents

	//opt := option.WithCredentialsFile("./modinfo.json")
	opt := option.WithCredentialsJSON([]byte(APIKEY))

	// Initialize the app with a service account, granting admin privileges
	app, err := firebase.NewApp(ctx, conf, opt)
	if err != nil {
		log.Fatalln("Error initializing app:", err)
	}

	client, err := app.Database(ctx)
	if err != nil {
		log.Fatalln("Error initializing database client:", err)
	}

	//const test = `json:"about_,omitempty"`

	type Data struct {
		Count int `json:"count,omitempty"`
	}

	fn := func(t db.TransactionNode) (interface{}, error) {
		var currentValue int
		if err := t.Unmarshal(&currentValue); err != nil {
			return nil, err
		}
		return currentValue + 1, nil
	}

	//pv
	println(strings.Replace(ip, ".", "_", -1))
	databasePath := fmt.Sprintf("modinfo_page_views/%s", path)
	ref := client.NewRef(databasePath)
	if err := ref.Transaction(ctx, fn); err != nil {
		log.Fatalln("Transaction failed to commit:", err)
	}
	var currentPV int
	ref.Get(ctx, &currentPV)
	//totalPV
	ref = client.NewRef("modinfo_page_views/total_page_views")
	if err := ref.Transaction(ctx, fn); err != nil {
		log.Fatalln("Transaction failed to commit:", err)
	}
	var currentTotalPV int
	ref.Get(ctx, &currentTotalPV)

	//ip
	databasePath = fmt.Sprintf("ip/%s", strings.Replace(ip, ".", "_", -1))
	ref = client.NewRef(databasePath)
	if err := ref.Transaction(ctx, fn); err != nil {
		log.Fatalln("Transaction failed to commit:", err)
	}

	return fmt.Sprintf(`{"pv":%d,"totalpv":%d}`, currentPV, currentTotalPV)
}
