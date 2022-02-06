package codersdk

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/hashicorp/yamux"
	"golang.org/x/xerrors"
	"nhooyr.io/websocket"

	"github.com/coder/coder/coderd"
	"github.com/coder/coder/provisionerd/proto"
	"github.com/coder/coder/provisionersdk"
)

func (c *Client) ProvisionerDaemons(ctx context.Context) ([]coderd.ProvisionerDaemon, error) {
	res, err := c.request(ctx, http.MethodGet, "/api/v2/provisioners/daemons", nil)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, readBodyAsError(res)
	}
	var daemons []coderd.ProvisionerDaemon
	return daemons, json.NewDecoder(res.Body).Decode(&daemons)
}

// ProvisionerDaemonClient returns the gRPC service for a provisioner daemon implementation.
func (c *Client) ProvisionerDaemonClient(ctx context.Context) (proto.DRPCProvisionerDaemonClient, error) {
	serverURL, err := c.URL.Parse("/api/v2/provisioners/daemons/serve")
	if err != nil {
		return nil, xerrors.Errorf("parse url: %w", err)
	}
	conn, res, err := websocket.Dial(ctx, serverURL.String(), &websocket.DialOptions{
		HTTPClient: c.httpClient,
		// Need to disable compression to avoid a data-race.
		CompressionMode: websocket.CompressionDisabled,
	})
	if err != nil {
		if res == nil {
			return nil, err
		}
		return nil, readBodyAsError(res)
	}
	config := yamux.DefaultConfig()
	config.LogOutput = io.Discard
	session, err := yamux.Client(websocket.NetConn(ctx, conn, websocket.MessageBinary), config)
	if err != nil {
		return nil, xerrors.Errorf("multiplex client: %w", err)
	}
	return proto.NewDRPCProvisionerDaemonClient(provisionersdk.Conn(session)), nil
}
