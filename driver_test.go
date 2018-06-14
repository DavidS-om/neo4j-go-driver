package neo4j_go_driver

import (
	"testing"
)

func assertNil(t *testing.T, err error) {
	if err != nil {
		println(err.Error())
		t.Error("An error occurred")
	}
}

func TestDriver(t *testing.T) {
	uri := "localhost:7687"
	driver, err := NewDriver(uri, NoAuth(), nil)
	assertNil(t, err)
	if driver.Target().String() != uri {
		t.Errorf("driver.uri = %q", uri)
	}
	//if driver.connection.ProtocolVersion != 2 {
	//	t.Errorf("Expected protocol v2")
	//}
}
