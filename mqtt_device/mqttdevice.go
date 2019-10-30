package mqttdevice

import (
	"fmt"
	MQTT "github.com/eclipse/paho.mqtt.golang"
	"log"
)

type Publisher interface {
	Connect()
	Close()
	Publish(topic string, payload interface{})
}

type PahoMqttPublisher struct {
	Uri      string
	Username string
	Password string
	ClientId string
	Oos      int
	Retain   bool
	client   MQTT.Client
}

// Publish message to broker
func (p *PahoMqttPublisher) Publish(topic string, payload interface{}) {
	tokenResp := p.client.Publish(topic, byte(p.Oos), p.Retain, payload)
	if tokenResp.Error() != nil {
		log.Fatalf("%+v\n", tokenResp.Error())
	}
}

// Close connection to broker
func (p *PahoMqttPublisher) Close() {
	p.client.Disconnect(500)
}

func (p *PahoMqttPublisher) Connect() {
	if p.client != nil && p.client.IsConnected() {
		return
	}
	//create a ClientOptions struct setting the broker address, clientid, turn
	//off trace output and set the default message handler
	opts := MQTT.NewClientOptions().AddBroker(p.Uri)
	opts.SetUsername(p.Username)
	opts.SetPassword(p.Password)
	opts.SetClientID(p.ClientId)
	opts.SetAutoReconnect(true)
	opts.SetDefaultPublishHandler(
		//define a function for the default message handler
		func(client MQTT.Client, msg MQTT.Message) {
			fmt.Printf("TOPIC: %s\n", msg.Topic())
			fmt.Printf("MSG: %s\n", msg.Payload())
		})

	//create and start a client using the above ClientOptions
	p.client = MQTT.NewClient(opts)
	if token := p.client.Connect(); token.Wait() && token.Error() != nil {
		panic(token.Error())
	}
}
