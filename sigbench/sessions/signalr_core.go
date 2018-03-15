package sessions

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/vmihailenco/msgpack"
)

const SignalRCoreTerminator = '\x1e'
const LatencyArrayLen int = 11
const LatencyStep int64 = 100

type SignalRCoreHandshakeResp struct {
	AvailableTransports []string `json:"availableTransports"`
	ConnectionId        string   `json:"connectionId"`
}

type SignalRCommon struct {
        Type         int      `json:"type"`
}

type SignalRCoreInvocation struct {
	//InvocationId string `json:"invocationId"`
	Type         int    `json:"type"`
	Target       string `json:"target"`
	//	NonBlocking  bool     `json:"nonBlocking"`
	Arguments []string `json:"arguments"`
}

type SignalRCoreServiceCompletion struct {
	InvocationId string            `json:"invocationId"`
	Type         int               `json:"type"`
	Meta         map[string]string `json:"meta"`
	Result       string            `json:"result"`
}

type MsgpackInvocation struct {
	MessageType  int32
	InvocationId string
	Target       string
	Arguments    []string
}

type MsgpackInvocationWithNonblocking struct {
	MessageType  int32
	InvocationId string
	NonBlocking  bool
	Target       string
	Arguments    []string
}

type ServiceMsgpackInvocationWithNonblocking struct {
	MessageType  int32
	InvocationId string
	NonBlocking  bool
	Target       string
	Arguments    []string
	//Meta map[string]string
}

func (m *ServiceMsgpackInvocationWithNonblocking) EncodeMsgpack(enc *msgpack.Encoder) error {
	enc.EncodeArrayLen(5)
	return enc.Encode(m.MessageType, m.InvocationId, m.NonBlocking, m.Target, m.Arguments)
}

func (m *ServiceMsgpackInvocationWithNonblocking) DecodeMsgpack(dec *msgpack.Decoder) error {
	dec.DecodeArrayLen()
	messageType, err := dec.DecodeInt32()
	if err != nil {
		fmt.Printf("Failed to decode message %v\n", dec)
		return err
	}
	m.MessageType = messageType
	if messageType == 1 {
		return dec.Decode(&m.InvocationId, &m.NonBlocking, &m.Target, &m.Arguments)
	}
	return nil
}

func (m *MsgpackInvocationWithNonblocking) EncodeMsgpack(enc *msgpack.Encoder) error {
	enc.EncodeArrayLen(5)
	return enc.Encode(m.MessageType, m.InvocationId, m.NonBlocking, m.Target, m.Arguments)
}

func (m *MsgpackInvocationWithNonblocking) DecodeMsgpack(dec *msgpack.Decoder) error {
	dec.DecodeArrayLen()
	messageType, err := dec.DecodeInt32()
	if err != nil {
		fmt.Printf("Failed to decode message %v\n", dec)
		return err
	}
	m.MessageType = messageType
	if messageType == 1 {
		return dec.Decode(&m.InvocationId, &m.NonBlocking, &m.Target, &m.Arguments)
	}
	return nil
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

func unmarshal2ServiceMsgpackContent(msg []byte, containNonBlocking bool) (int32, string, []string, error) {
	if containNonBlocking {
		var content ServiceMsgpackInvocationWithNonblocking
		err := msgpack.Unmarshal(msg, &content)
		if err != nil {
			return 0, "", nil, err
		}
		return content.MessageType, content.Target, content.Arguments, nil
	} else {
		/*
			var content MsgpackInvocation
			err := msgpack.Unmarshal(msg, &content)
			if err != nil {
				return 0, "", nil, err
			}
			return content.MessageType, content.Target, content.Arguments, nil
		*/
		return 0, "", nil, nil
	}
}

func unmarshal2MsgpackContent(msg []byte, containNonBlocking bool) (int32, string, []string, error) {
	if containNonBlocking {
		var content MsgpackInvocationWithNonblocking
		err := msgpack.Unmarshal(msg, &content)
		if err != nil {
			return 0, "", nil, err
		}
		return content.MessageType, content.Target, content.Arguments, nil
	} else {
		var content MsgpackInvocation
		err := msgpack.Unmarshal(msg, &content)
		if err != nil {
			return 0, "", nil, err
		}
		return content.MessageType, content.Target, content.Arguments, nil
	}
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

func TokenizeBytes(data []byte) ([][]byte) {
	return bytes.Split(data, []byte{0x1e});
}
