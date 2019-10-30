package main

import (
	"context"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"testing"
	"time"
	"warmup4ie2mqtt/warmup4ie"
)

func startMqttContainer(t *testing.T) (context.Context, testcontainers.Container, error) {
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
	return ctx, mqttC, err
}

type thermostatMock struct{}

func (t *thermostatMock) ListLocations() (*[]warmup4ie.Location, error) {
	panic("implement me")
}

func (t *thermostatMock) ListRooms() (*[]warmup4ie.Room, error) {
	return &[]warmup4ie.Room{
		{
			Id:      1,
			Name:    "Room1",
			RunMode: warmup4ie.RunModeFixed,
			TargetTemp: warmup4ie.Temperature{
				RawTemperature: 220,
			},
			CurrentTemp: warmup4ie.Temperature{
				RawTemperature: 190,
			},
			Thermostat4IES: nil,
		},
		{
			Id:      2,
			Name:    "Room2",
			RunMode: warmup4ie.RunModeForced,
			TargetTemp: warmup4ie.Temperature{
				RawTemperature: 250,
			},
			CurrentTemp: warmup4ie.Temperature{
				RawTemperature: 200,
			},
			Thermostat4IES: nil,
		},
	}, nil
}

type fakePublisher struct {
	msg map[string]interface{}
}

func (f fakePublisher) Connect() {
	panic("implement me")
}

func (f fakePublisher) Close() {
	panic("implement me")
}

func (f fakePublisher) Publish(topic string, payload interface{}) {
	f.msg[topic] = payload
}

func TestMonitorDevice(t *testing.T) {
	th := thermostatMock{}
	p := fakePublisher{}
	p.msg = make(map[string]interface{})

	go MonitorDevice(&th, p, "room", 1*time.Millisecond)
	time.Sleep(10 * time.Millisecond)
	if len(p.msg) != 4 {
		t.Errorf("4 messages are expected, pusblished: %d", len(p.msg))
	}

	expectedTopic := map[string]string{
		"room/room1/temperature/floor":        "19.0",
		"room/room1/temperature/floor/target": "22.0",
		"room/room2/temperature/floor":        "20.0",
		"room/room2/temperature/floor/target": "25.0",
	}
	for topic, temp := range expectedTopic {
		if p.msg[topic] == nil {
			t.Errorf("No temperature published on topic %s", topic)
		}
		if p.msg[topic] != temp {
			t.Errorf("Bad temperature for topic %s. expected %s but received %v", topic, temp, p.msg[topic])
		}
	}
}
