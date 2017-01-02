package meetup_client_go

import "testing"

func TestNewMeetupClient(t *testing.T) {
	t.Log("Testing New...")
	key := ""
	t.Log("API key empty. Should fail.")
	c := NewMeetupClient(key)
	if c != nil {
		t.Errorf("Expected failure for empty key. Key:%s", key)
	}
	key = "abcde"
	c = NewMeetupClient(key)
	if c == nil {
		t.Errorf("Expected success for random text key. Returned nil. Key:%s", key)
	}
}

func TestMeetupClient_GetEvents(t *testing.T) {

}
