package main

import (
	"flag"
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	mqttdevice "warmup4ie2mqtt/mqtt_device"
	"warmup4ie2mqtt/warmup4ie"
)

const DefaultClientId = "warmup4ie2zwave"

func MonitorDevice(t warmup4ie.Thermostat, p mqttdevice.Publisher, topicBase string, idleTime time.Duration) {
	var token MQTT.Token
	for {
		if rooms, err := t.ListRooms(); err != nil {
			log.Fatalf("%+v\n", token.Error())
		} else {
			for _, room := range *rooms {
				topic := fmt.Sprintf("%s/%s/temperature/floor", topicBase, strings.ToLower(room.Name))
				p.Publish(topic, fmt.Sprintf("%.1f", room.CurrentTemp.GetValue()))
				topic = fmt.Sprintf("%s/%s/temperature/floor/target", topicBase, strings.ToLower(room.Name))
				p.Publish(topic, fmt.Sprintf("%.1f", room.TargetTemp.GetValue()))
			}
		}
		time.Sleep(idleTime)
	}
}

func main() {
	var mqttBroker, qos, clientId, topicBase, wEmail, wPassword string
	setDefaultValueFromEnv(&clientId, "MQTT_CLIENT_ID", DefaultClientId)
	setDefaultValueFromEnv(&mqttBroker, "MQTT_BROKER", "tcp://127.0.0.1:1883")
	setDefaultValueFromEnv(&qos, "MQTT_QOS", "0")
	mqttQos, err := strconv.Atoi(qos)
	if err != nil {
		log.Panicf("invalid mqtt qos value: %v", qos)
	}
	_, mqttRetain := os.LookupEnv("MQTT_RETAIN")

	publisher := mqttdevice.PahoMqttPublisher{}
	flag.StringVar(&publisher.Uri, "mqtt-broker", mqttBroker, "Broker Uri, use MQTT_BROKER env if arg not set")
	flag.StringVar(&publisher.Username, "mqtt-username", os.Getenv("MQTT_USERNAME"), "Broker Username, use MQTT_USERNAME env if arg not set")
	flag.StringVar(&publisher.Password, "mqtt-password", os.Getenv("MQTT_PASSWORD"), "Broker Password, MQTT_PASSWORD env if args not set")
	flag.StringVar(&publisher.ClientId, "mqtt-client-id", clientId, "Mqtt client id, use MQTT_CLIENT_ID env if args not set")
	flag.IntVar(&publisher.Oos, "mqtt-qos", mqttQos, "Qos to pusblish message, use MQTT_QOS env if arg not set")
	flag.StringVar(&topicBase, "mqtt-topic-base", os.Getenv("MQTT_TOPIC_BASE"), "Mqtt topic prefix, use MQTT_TOPIC_BASE if args not set")
	flag.BoolVar(&publisher.Retain, "mqtt-retain", mqttRetain, "Retain mqtt message, if not set, true if MQTT_RETAIN env variable is set")
	flag.StringVar(&wEmail, "warmup-email", os.Getenv("WARMUP_EMAIL"), "Warmup email used to logon, use WARMUP_USERNAME env if arg not set")
	flag.StringVar(&wPassword, "warmup-password", os.Getenv("WARMUP_PASSWORD"), "Warmup password used to logon, use WARMUP_PASSWORD env if arg not set")

	flag.Parse()
	if len(os.Args) <= 1 {
		flag.PrintDefaults()
		os.Exit(1)
	}

	publisher.Connect()
	defer publisher.Close()
	device, err := warmup4ie.NewDevice(wEmail, wPassword)
	if err != nil {
		log.Panicf("unable to connect to warmup server: %v\n", err)
	}
	MonitorDevice(device, &publisher, topicBase, 1*time.Second)
}

func setDefaultValueFromEnv(value *string, key string, defaultValue string) {
	if os.Getenv(key) != "" {
		*value = os.Getenv(key)
	} else {
		*value = defaultValue
	}
}
