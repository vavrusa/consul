package consul

import (
	"os"
	"testing"
	"time"

	"github.com/hashicorp/consul/acl"
	"github.com/hashicorp/consul/agent/structs"
	"github.com/hashicorp/consul/testrpc"
	"github.com/hashicorp/net-rpc-msgpackrpc"
	"github.com/stretchr/testify/assert"
)

// Test basic creation
func TestIntentionApply_new(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Setup a basic record to create
	ixn := structs.IntentionRequest{
		Datacenter: "dc1",
		Op:         structs.IntentionOpCreate,
		Intention: &structs.Intention{
			SourceNS:        structs.IntentionDefaultNamespace,
			SourceName:      "test",
			DestinationNS:   structs.IntentionDefaultNamespace,
			DestinationName: "test",
			Action:          structs.IntentionActionAllow,
			SourceType:      structs.IntentionSourceConsul,
			Meta:            map[string]string{},
		},
	}
	var reply string

	// Record now to check created at time
	now := time.Now()

	// Create
	assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply))
	assert.NotEmpty(reply)

	// Read
	ixn.Intention.ID = reply
	{
		req := &structs.IntentionQueryRequest{
			Datacenter:  "dc1",
			IntentionID: ixn.Intention.ID,
		}
		var resp structs.IndexedIntentions
		assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Get", req, &resp))
		assert.Len(resp.Intentions, 1)
		actual := resp.Intentions[0]
		assert.Equal(resp.Index, actual.ModifyIndex)
		assert.WithinDuration(now, actual.CreatedAt, 5*time.Second)
		assert.WithinDuration(now, actual.UpdatedAt, 5*time.Second)

		actual.CreateIndex, actual.ModifyIndex = 0, 0
		actual.CreatedAt = ixn.Intention.CreatedAt
		actual.UpdatedAt = ixn.Intention.UpdatedAt
		assert.Equal(ixn.Intention, actual)
	}
}

// Test the source type defaults
func TestIntentionApply_defaultSourceType(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Setup a basic record to create
	ixn := structs.IntentionRequest{
		Datacenter: "dc1",
		Op:         structs.IntentionOpCreate,
		Intention: &structs.Intention{
			SourceNS:        structs.IntentionDefaultNamespace,
			SourceName:      "test",
			DestinationNS:   structs.IntentionDefaultNamespace,
			DestinationName: "test",
			Action:          structs.IntentionActionAllow,
		},
	}
	var reply string

	// Create
	assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply))
	assert.NotEmpty(reply)

	// Read
	ixn.Intention.ID = reply
	{
		req := &structs.IntentionQueryRequest{
			Datacenter:  "dc1",
			IntentionID: ixn.Intention.ID,
		}
		var resp structs.IndexedIntentions
		assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Get", req, &resp))
		assert.Len(resp.Intentions, 1)
		actual := resp.Intentions[0]
		assert.Equal(structs.IntentionSourceConsul, actual.SourceType)
	}
}

// Shouldn't be able to create with an ID set
func TestIntentionApply_createWithID(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Setup a basic record to create
	ixn := structs.IntentionRequest{
		Datacenter: "dc1",
		Op:         structs.IntentionOpCreate,
		Intention: &structs.Intention{
			ID:         generateUUID(),
			SourceName: "test",
		},
	}
	var reply string

	// Create
	err := msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply)
	assert.NotNil(err)
	assert.Contains(err, "ID must be empty")
}

// Test basic updating
func TestIntentionApply_updateGood(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Setup a basic record to create
	ixn := structs.IntentionRequest{
		Datacenter: "dc1",
		Op:         structs.IntentionOpCreate,
		Intention: &structs.Intention{
			SourceNS:        structs.IntentionDefaultNamespace,
			SourceName:      "test",
			DestinationNS:   structs.IntentionDefaultNamespace,
			DestinationName: "test",
			Action:          structs.IntentionActionAllow,
			SourceType:      structs.IntentionSourceConsul,
			Meta:            map[string]string{},
		},
	}
	var reply string

	// Create
	assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply))
	assert.NotEmpty(reply)

	// Read CreatedAt
	var createdAt time.Time
	ixn.Intention.ID = reply
	{
		req := &structs.IntentionQueryRequest{
			Datacenter:  "dc1",
			IntentionID: ixn.Intention.ID,
		}
		var resp structs.IndexedIntentions
		assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Get", req, &resp))
		assert.Len(resp.Intentions, 1)
		actual := resp.Intentions[0]
		createdAt = actual.CreatedAt
	}

	// Sleep a bit so that the updated at will definitely be different, not much
	time.Sleep(1 * time.Millisecond)

	// Update
	ixn.Op = structs.IntentionOpUpdate
	ixn.Intention.ID = reply
	ixn.Intention.SourceName = "bar"
	assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply))

	// Read
	ixn.Intention.ID = reply
	{
		req := &structs.IntentionQueryRequest{
			Datacenter:  "dc1",
			IntentionID: ixn.Intention.ID,
		}
		var resp structs.IndexedIntentions
		assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Get", req, &resp))
		assert.Len(resp.Intentions, 1)
		actual := resp.Intentions[0]
		assert.Equal(createdAt, actual.CreatedAt)
		assert.WithinDuration(time.Now(), actual.UpdatedAt, 5*time.Second)

		actual.CreateIndex, actual.ModifyIndex = 0, 0
		actual.CreatedAt = ixn.Intention.CreatedAt
		actual.UpdatedAt = ixn.Intention.UpdatedAt
		assert.Equal(ixn.Intention, actual)
	}
}

// Shouldn't be able to update a non-existent intention
func TestIntentionApply_updateNonExist(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Setup a basic record to create
	ixn := structs.IntentionRequest{
		Datacenter: "dc1",
		Op:         structs.IntentionOpUpdate,
		Intention: &structs.Intention{
			ID:         generateUUID(),
			SourceName: "test",
		},
	}
	var reply string

	// Create
	err := msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply)
	assert.NotNil(err)
	assert.Contains(err, "Cannot modify non-existent intention")
}

// Test basic deleting
func TestIntentionApply_deleteGood(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Setup a basic record to create
	ixn := structs.IntentionRequest{
		Datacenter: "dc1",
		Op:         structs.IntentionOpCreate,
		Intention: &structs.Intention{
			SourceNS:        "test",
			SourceName:      "test",
			DestinationNS:   "test",
			DestinationName: "test",
			Action:          structs.IntentionActionAllow,
		},
	}
	var reply string

	// Create
	assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply))
	assert.NotEmpty(reply)

	// Delete
	ixn.Op = structs.IntentionOpDelete
	ixn.Intention.ID = reply
	assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply))

	// Read
	ixn.Intention.ID = reply
	{
		req := &structs.IntentionQueryRequest{
			Datacenter:  "dc1",
			IntentionID: ixn.Intention.ID,
		}
		var resp structs.IndexedIntentions
		err := msgpackrpc.CallWithCodec(codec, "Intention.Get", req, &resp)
		assert.NotNil(err)
		assert.Contains(err, ErrIntentionNotFound.Error())
	}
}

// Test apply with a deny ACL
func TestIntentionApply_aclDeny(t *testing.T) {
	t.Parallel()
	dir1, s1 := testServerWithConfig(t, func(c *Config) {
		c.ACLDatacenter = "dc1"
		c.ACLMasterToken = "root"
		c.ACLDefaultPolicy = "deny"
	})
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Create an ACL with write permissions
	var token string
	{
		var rules = `
service "foo" {
	policy = "deny"
	intentions = "write"
}`

		req := structs.ACLRequest{
			Datacenter: "dc1",
			Op:         structs.ACLSet,
			ACL: structs.ACL{
				Name:  "User token",
				Type:  structs.ACLTypeClient,
				Rules: rules,
			},
			WriteRequest: structs.WriteRequest{Token: "root"},
		}
		if err := msgpackrpc.CallWithCodec(codec, "ACL.Apply", &req, &token); err != nil {
			t.Fatalf("err: %v", err)
		}
	}

	// Setup a basic record to create
	ixn := structs.IntentionRequest{
		Datacenter: "dc1",
		Op:         structs.IntentionOpCreate,
		Intention:  structs.TestIntention(t),
	}
	ixn.Intention.DestinationName = "foobar"

	// Create without a token should error since default deny
	var reply string
	err := msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply)
	if !acl.IsErrPermissionDenied(err) {
		t.Fatalf("bad: %v", err)
	}

	// Now add the token and try again.
	ixn.WriteRequest.Token = token
	if err = msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply); err != nil {
		t.Fatalf("err: %v", err)
	}

	// Read
	ixn.Intention.ID = reply
	{
		req := &structs.IntentionQueryRequest{
			Datacenter:  "dc1",
			IntentionID: ixn.Intention.ID,
		}
		var resp structs.IndexedIntentions
		if err := msgpackrpc.CallWithCodec(codec, "Intention.Get", req, &resp); err != nil {
			t.Fatalf("err: %v", err)
		}
		if len(resp.Intentions) != 1 {
			t.Fatalf("bad: %v", resp)
		}
		actual := resp.Intentions[0]
		if resp.Index != actual.ModifyIndex {
			t.Fatalf("bad index: %d", resp.Index)
		}

		actual.CreateIndex, actual.ModifyIndex = 0, 0
		actual.CreatedAt = ixn.Intention.CreatedAt
		actual.UpdatedAt = ixn.Intention.UpdatedAt
		if !reflect.DeepEqual(actual, ixn.Intention) {
			t.Fatalf("bad:\n\n%#v\n\n%#v", actual, ixn.Intention)
		}
	}
}

func TestIntentionList(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()

	codec := rpcClient(t, s1)
	defer codec.Close()
	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Test with no intentions inserted yet
	{
		req := &structs.DCSpecificRequest{
			Datacenter: "dc1",
		}
		var resp structs.IndexedIntentions
		assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.List", req, &resp))
		assert.NotNil(resp.Intentions)
		assert.Len(resp.Intentions, 0)
	}
}

// Test basic matching. We don't need to exhaustively test inputs since this
// is tested in the agent/consul/state package.
func TestIntentionMatch_good(t *testing.T) {
	t.Parallel()

	assert := assert.New(t)
	dir1, s1 := testServer(t)
	defer os.RemoveAll(dir1)
	defer s1.Shutdown()
	codec := rpcClient(t, s1)
	defer codec.Close()

	testrpc.WaitForLeader(t, s1.RPC, "dc1")

	// Create some records
	{
		insert := [][]string{
			{"foo", "*"},
			{"foo", "bar"},
			{"foo", "baz"}, // shouldn't match
			{"bar", "bar"}, // shouldn't match
			{"bar", "*"},   // shouldn't match
			{"*", "*"},
		}

		for _, v := range insert {
			ixn := structs.IntentionRequest{
				Datacenter: "dc1",
				Op:         structs.IntentionOpCreate,
				Intention: &structs.Intention{
					SourceNS:        "default",
					SourceName:      "test",
					DestinationNS:   v[0],
					DestinationName: v[1],
					Action:          structs.IntentionActionAllow,
				},
			}

			// Create
			var reply string
			assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Apply", &ixn, &reply))
		}
	}

	// Match
	req := &structs.IntentionQueryRequest{
		Datacenter: "dc1",
		Match: &structs.IntentionQueryMatch{
			Type: structs.IntentionMatchDestination,
			Entries: []structs.IntentionMatchEntry{
				{
					Namespace: "foo",
					Name:      "bar",
				},
			},
		},
	}
	var resp structs.IndexedIntentionMatches
	assert.Nil(msgpackrpc.CallWithCodec(codec, "Intention.Match", req, &resp))
	assert.Len(resp.Matches, 1)

	expected := [][]string{{"foo", "bar"}, {"foo", "*"}, {"*", "*"}}
	var actual [][]string
	for _, ixn := range resp.Matches[0] {
		actual = append(actual, []string{ixn.DestinationNS, ixn.DestinationName})
	}
	assert.Equal(expected, actual)
}
