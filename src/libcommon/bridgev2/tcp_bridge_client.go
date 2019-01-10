package bridgev2

import (
    "util/logger"
    "strconv"
    "crypto/md5"
    "errors"
    "app"
    "util/json"
    "libcommon"
)

var connPool *libcommon.ClientConnectionPool

type BridgeClient struct {
    // storage server info
    server *app.ServerInfo
    connManager *ConnectionManager
    // connect state
    // 0: not connect
    // 1: connected but not validate
    // 2: validated
    // 3: disconnected
    state int
}

func init() {
    connPool = &libcommon.ClientConnectionPool{}
    connPool.Init(50)
}

// create a new instance for bridgev2.Server
func NewClient(server *app.ServerInfo) *BridgeClient {
    return &BridgeClient {server, nil, 0}
}


func (client *BridgeClient) Close() {
    if client.connManager != nil {
        connPool.ReturnBrokenConnBridge(client.server, client.connManager.Conn)
    }
}

func (client *BridgeClient) Destroy() {
    if client.connManager != nil {
        connPool.ReturnConnBridge(client.server, client.connManager.Conn)
    }
}


func (client *BridgeClient) Connect() error {
    if client.state > 0 {
        panic(errors.New("already connected"))
    }
    conn, err := connPool.GetConn(client.server)
    if err != nil {
        return err
    }
    h, p := client.server.GetHostAndPortByAccessFlag()
    logger.Debug("connect to", h + ":" + strconv.Itoa(p), "success")
    client.connManager = &ConnectionManager{
        Conn: conn,
        Side: CLIENT_SIDE,
        Md: md5.New(),
    }
    client.state = 1
    return nil
}


func (client *BridgeClient) Validate() (*ConnectResponseMeta, error) {
    client.requireStatus(1)
    meta := &ConnectMeta{
        Secret: app.SECRET,
        UUID: app.UUID,
    }

    frame := &Frame{}
    frame.SetOperation(FRAME_OPERATION_VALIDATE)
    frame.SetMeta(meta)
    frame.SetMetaBodyLength(0)
    if err := client.connManager.Send(frame); err != nil {
        return nil, err
    }
    response, e := client.connManager.Receive()
    if e != nil {
        return nil, e
    }
    if response != nil {
        var res = &ConnectResponseMeta{}
        e := json.Unmarshal(response.frameMeta, res)
        if e != nil {
            return nil, e
        }
        return res, nil
    } else {
        return nil, errors.New("receive empty response from server")
    }
}

func (client *BridgeClient) requireStatus(requiredState int) error {
    if client.state < requiredState {
        panic(errors.New("connect state not satisfied, expect " + strconv.Itoa(requiredState) + ", now is " + strconv.Itoa(client.state)))
    }
    return nil
}





