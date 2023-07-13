package main

import (
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gin-gonic/gin"
	"log"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/viper"
)

var config = viper.New()

var mqttMessageHandler mqtt.MessageHandler = func(client mqtt.Client, msg mqtt.Message) {
	fmt.Printf("TOPIC: %s\n", msg.Topic())
	fmt.Printf("MSG: %s\n", msg.Payload())
	fmt.Println("----")
}

var mqttConnLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	println("lost mqtt connection, reconnecting")
	time.Sleep(3 * time.Second)
	client.Connect()
}

func main() {
	go func() {
		r := gin.Default()
		//r.GET("/ping", func(c *gin.Context) {
		//	c.JSON(http.StatusOK, gin.H{
		//		"message": "pong",
		//	})
		//})
		r.LoadHTMLGlob("templates/*")
		r.GET("/index", func(c *gin.Context) {
			var template, _ = os.ReadFile(("msg_template/inhand.json"))
			//var templateBuity, _ = json.MarshalIndent(string(template), "", "  ")
			c.HTML(http.StatusOK, "index.tmpl", gin.H{
				"template": string(template),
			})
		})

		r.Run("0.0.0.0:8080") // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
	}()
	initConf()

	mqttAddress := getConf("mqtt.address")
	mqttClientId := getConf("mqtt.client_id")
	mqttUser := getConf("mqtt.user")
	mqttPass := getConf("mqtt.pass")
	subTopic := getConf("mqtt.sub_topic")
	pubTopic := getConf("mqtt.pub_topic")

	//mqttUser = ""
	//mqttPass = ""

	//mqtt.DEBUG = log.New(os.Stdout, "", 0)
	mqtt.ERROR = log.New(os.Stdout, "", 0)
	opts := mqtt.NewClientOptions().AddBroker("tcp://" + mqttAddress).SetClientID(mqttClientId).SetUsername(mqttUser).SetPassword(mqttPass)

	opts.SetKeepAlive(60 * time.Second)
	// Set the message callback handler
	opts.SetDefaultPublishHandler(mqttMessageHandler)
	opts.SetConnectionLostHandler(mqttConnLostHandler)
	opts.SetPingTimeout(1 * time.Second)

	c := mqtt.NewClient(opts)
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}

	// Subscribe to a topic
	if token := c.Subscribe(subTopic, 0, nil); token.Wait() && token.Error() != nil {
		fmt.Println(token.Error())
		os.Exit(1)
	}

	// Publish a message
	for true {
		if !c.IsConnected() {
			break
		}
		content := getMsg()
		token := c.Publish(pubTopic, 0, false, content)
		token.Wait()
		time.Sleep(3 * time.Second)
	}

	// Unscribe
	//if token := c.Unsubscribe("testtopic/#"); token.Wait() && token.Error() != nil {
	//	fmt.Println(token.Error())
	//	os.Exit(1)
	//}

	// Disconnect
	//c.Disconnect(250)
	//time.Sleep(1 * time.Second)

}

func getMsg() string {
	content, _ := os.ReadFile("msg_template/inhand.json")
	msg := string(content)
	randCount := strings.Count(msg, "{{rand}}")
	for i := 1; i <= randCount; i++ {
		msg = strings.Replace(msg, "{{rand}}", fmt.Sprintf("%6.2f", rand.Float64()*1000), 1)
	}
	msg = strings.Replace(msg, "{{ts}}", strconv.FormatInt(time.Now().Unix(), 10), 2)
	return msg
}

func initConf() {
	root, err := os.Getwd()
	if err != nil {
		panic(err)
	}
	config.AddConfigPath(root)
	config.SetConfigName("config")
	config.SetConfigType("yaml")

}

func getConf(path string) string {
	//尝试进行配置读取
	if err := config.ReadInConfig(); err != nil {
		panic(err)
	}

	return config.Get(path).(string)
}
