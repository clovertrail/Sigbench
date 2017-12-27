package sessions

import (
	"encoding/json"
	"fmt"

	"github.com/vmihailenco/msgpack"
)
const SignalRCoreTerminator = '\x1e'

type SignalRCoreHandshakeResp struct {
	AvailableTransports []string `json:"availableTransports"`
	ConnectionId        string   `json:"connectionId"`
}

type SignalRCoreInvocation struct {
	InvocationId string   `json:"invocationId"`
	Type         int      `json:"type"`
	Target       string   `json:"target"`
//	NonBlocking  bool     `json:"nonBlocking"`
	Arguments    []string `json:"arguments"`
}

type MsgpackInvocation struct {
	MessageType  int32
	InvocationId string
	Target       string
	Arguments    []string
}

func (m *MsgpackInvocation) EncodeMsgpack(enc *msgpack.Encoder) error {
	enc.EncodeArrayLen(4)
	return enc.Encode(m.MessageType, m.InvocationId, m.Target, m.Arguments)
}

func (m *MsgpackInvocation) DecodeMsgpack(dec *msgpack.Decoder) error {
	dec.DecodeArrayLen()
	messageType, err := dec.DecodeInt32()
	if err != nil {
		fmt.Printf("Failed to decode message %v\n", dec)
		return err
	}
	m.MessageType = messageType
	if messageType == 1 {
		return dec.Decode(&m.InvocationId, &m.Target, &m.Arguments)
	}
	return nil
}

func encodeSignalRBinary(bytes []byte) ([]byte, error) {
        buffer := make([]byte, 0, 5+len(bytes))
        length := len(bytes)
        for length > 0 {
                current := byte(length & 0x7F)
                length >>= 7
                if length > 0 {
                        current |= 0x80
                }
                buffer = append(buffer, current)
        }
        if len(buffer) == 0 {
                buffer = append(buffer, 0)
        }
        buffer = append(buffer, bytes...)
        return buffer, nil
}

var numBitsToShift = []uint{0, 7, 14, 21, 28}

func decodeSignalRBinary(bytes []byte) ([]byte, error) {
        moreBytes := true
        msgLen := 0
        numBytes := 0
        for moreBytes && numBytes < len(bytes) {
                byteRead := bytes[numBytes]
                msgLen = msgLen | int(uint(byteRead&0x7F)<<numBitsToShift[numBytes])
                numBytes++
                moreBytes = (byteRead & 0x80) != 0
        }

        if msgLen+numBytes > len(bytes) {
                return nil, fmt.Errorf("Not enough data in message, message length = %d, length section bytes = %d, data length = %d", msgLen, numBytes, len(bytes))
        }
        return bytes[numBytes : numBytes+msgLen], nil
}

func SerializeSignalRCoreMessage(body interface{}) ([]byte, error) {
	msg, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return append(msg, SignalRCoreTerminator), nil
}
