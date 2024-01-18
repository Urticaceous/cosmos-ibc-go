package host

import "fmt"

const (
	KeySequencePrefix                     = "sequences"
	KeyNextSeqSendPrefix                  = "nextSequenceSend"
	KeyNextSeqRecvPrefix                  = "nextSequenceRecv"
	KeyNextSeqAckPrefix                   = "nextSequenceAck"
	KeyPacketCommitmentPrefix             = "commitments"
	KeyPacketAckPrefix                    = "acks"
	KeyPacketReceiptPrefix                = "receipts"
	KeyCounterpartyNextSequenceSendPrefix = "counterpartyNextSequenceSend"
	KeyPruningSequenceStart               = "pruningSequenceStart"
	KeyPruningSequenceEnd                 = "pruningSequenceEnd"
)

// ICS04
// The following paths are the keys to the store as defined in https://github.com/cosmos/ibc/tree/master/spec/core/ics-004-channel-and-packet-semantics#store-paths

// NextSequenceSendPath defines the next send sequence counter store path
func NextSequenceSendPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyNextSeqSendPrefix, channelPath(portID, channelID))
}

// NextSequenceSendKey returns the store key for the send sequence of a particular
// channel binded to a specific port.
func NextSequenceSendKey(portID, channelID string) []byte {
	return []byte(NextSequenceSendPath(portID, channelID))
}

// NextSequenceRecvPath defines the next receive sequence counter store path.
func NextSequenceRecvPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyNextSeqRecvPrefix, channelPath(portID, channelID))
}

// NextSequenceRecvKey returns the store key for the receive sequence of a particular
// channel binded to a specific port
func NextSequenceRecvKey(portID, channelID string) []byte {
	return []byte(NextSequenceRecvPath(portID, channelID))
}

// NextSequenceAckPath defines the next acknowledgement sequence counter store path
func NextSequenceAckPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyNextSeqAckPrefix, channelPath(portID, channelID))
}

// NextSequenceAckKey returns the store key for the acknowledgement sequence of
// a particular channel binded to a specific port.
func NextSequenceAckKey(portID, channelID string) []byte {
	return []byte(NextSequenceAckPath(portID, channelID))
}

// PacketCommitmentPath defines the commitments to packet data fields store path
func PacketCommitmentPath(portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%s/%d", PacketCommitmentPrefixPath(portID, channelID), sequence)
}

// PacketCommitmentKey returns the store key of under which a packet commitment
// is stored
func PacketCommitmentKey(portID, channelID string, sequence uint64) []byte {
	return []byte(PacketCommitmentPath(portID, channelID, sequence))
}

// PacketCommitmentPrefixPath defines the prefix for commitments to packet data fields store path.
func PacketCommitmentPrefixPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s", KeyPacketCommitmentPrefix, channelPath(portID, channelID), KeySequencePrefix)
}

// PacketAcknowledgementPath defines the packet acknowledgement store path
func PacketAcknowledgementPath(portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%s/%d", PacketAcknowledgementPrefixPath(portID, channelID), sequence)
}

// PacketAcknowledgementKey returns the store key of under which a packet
// acknowledgement is stored
func PacketAcknowledgementKey(portID, channelID string, sequence uint64) []byte {
	return []byte(PacketAcknowledgementPath(portID, channelID, sequence))
}

// PacketAcknowledgementPrefixPath defines the prefix for commitments to packet data fields store path.
func PacketAcknowledgementPrefixPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s/%s", KeyPacketAckPrefix, channelPath(portID, channelID), KeySequencePrefix)
}

// PacketReceiptPath defines the packet receipt store path
func PacketReceiptPath(portID, channelID string, sequence uint64) string {
	return fmt.Sprintf("%s/%s/%s", KeyPacketReceiptPrefix, channelPath(portID, channelID), sequencePath(sequence))
}

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func PacketReceiptKey(portID, channelID string, sequence uint64) []byte {
	return []byte(PacketReceiptPath(portID, channelID, sequence))
}

// PacketReceiptPath defines the packet receipt store path
func CounterpartyNextSequenceSendPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyCounterpartyNextSequenceSendPrefix, channelPath(portID, channelID))
}

// PacketReceiptKey returns the store key of under which a packet
// receipt is stored
func CounterpartyNextSequenceSendKey(portID, channelID string) []byte {
	return []byte(CounterpartyNextSequenceSendPath(portID, channelID))
}

// PruningSequenceStartPath defines the path under which the pruning sequence starting value is stored
func PruningSequenceStartPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyPruningSequenceStart, channelPath(portID, channelID))
}

// PruningSequenceStartKey returns the store key for the pruning sequence start of a particular channel
func PruningSequenceStartKey(portID, channelID string) []byte {
	return []byte(PruningSequenceStartPath(portID, channelID))
}

// PruningSequenceEndPath defines the path under which the pruning sequence end is stored
func PruningSequenceEndPath(portID, channelID string) string {
	return fmt.Sprintf("%s/%s", KeyPruningSequenceEnd, channelPath(portID, channelID))
}

// PruningSequenceEndKey returns the store key for the pruning sequence end of a particular channel
func PruningSequenceEndKey(portID, channelID string) []byte {
	return []byte(PruningSequenceEndPath(portID, channelID))
}

func sequencePath(sequence uint64) string {
	return fmt.Sprintf("%s/%d", KeySequencePrefix, sequence)
}
