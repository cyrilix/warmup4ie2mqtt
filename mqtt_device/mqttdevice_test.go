package mqttdevice

import (
	"context"
	"fmt"
	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"testing"
	"time"
)

func startMqttContainer(t *testing.T) (context.Context, testcontainers.Container, string) {
	ctx := context.Background()
	req := testcontainers.ContainerRequest{
		Image:        "cyrilix/rabbitmq-mqtt",
		ExposedPorts: []string{"1883/tcp"},
		WaitingFor:   wait.ForLog("Server startup complete").WithPollInterval(1 * time.Second),
	}
	mqttC, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Error(err)
	}
	ip, err := mqttC.Host(ctx)
	if err != nil {
		t.Error(err)
	}
	port, err := mqttC.MappedPort(ctx, "1883/tcp")
	if err != nil {
		t.Error(err)
	}

	mqttUri := fmt.Sprintf("tcp://%s:%d", ip, port.Int())
	return ctx, mqttC, mqttUri
}

func TestIntegration(t *testing.T) {

	ctx, mqttC, mqttUri := startMqttContainer(t)
	defer mqttC.Terminate(ctx)

	t.Run("ConnectAndClose", func(t *testing.T) {
		t.Logf("Mqtt connection %s ready", mqttUri)

		p := PahoMqttPublisher{Uri: mqttUri, ClientId: "TestMqtt", Username: "guest", Password: "guest"}
		p.Connect()
		p.Close()
	})
	t.Run("Publish", func(t *testing.T) {
		options := mqtt.NewClientOptions().AddBroker(mqttUri)
		options.SetUsername("guest")
		options.SetPassword("guest")

		client := mqtt.NewClient(options)
		token := client.Connect()
		defer client.Disconnect(100)
		token.Wait()
		if token.Error() != nil {
			t.Fatalf("unable to connect to mqtt broker: %v\n", token.Error())
		}

		c := make(chan string)
		defer close(c)
		client.Subscribe("test/publish", 0, func(client mqtt.Client, message mqtt.Message) {
			c <- string(message.Payload())
		}).Wait()

		p := PahoMqttPublisher{Uri: mqttUri, ClientId: "TestMqtt", Username: "guest", Password: "guest"}
		p.Connect()
		defer p.Close()

		p.Publish("test/publish", "Test1234")
		result := <-c
		if result != "Test1234" {
			t.Fatalf("bad message: %v\n", result)
		}

	})
}
