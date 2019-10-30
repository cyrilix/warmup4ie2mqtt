package warmup4ie

import "testing"

const email = "warmup@cyrilix.fr"
const password = "$6Ypy'<;Hb+ZUu2"

func init2Thermostat(t *testing.T) *Device {
	device, err := NewDevice(email, password)
	if err != nil || device == nil {
		t.Errorf("unexpected error: %v", err)
	}
	return device
}

func TestDevice(t *testing.T) {
	device, err := NewDevice(email, password)
	if err != nil || device == nil {
		t.Errorf("unexpected error: %v", err)
	}

	device.ListRooms()
}
