package pcap

import (
	"encoding/binary"

	"github.com/google/gopacket/layers"

	"gitlab.x.lan/yunshan/droplet-libs/datatype"
)

const (
	MAX_PACKET_LEN  = 128
	MAC_ADDRESS_LEN = 6
	IP_ADDRESS_LEN  = 4
)

type RawPacket []byte

func NewRawPacket(buffer []byte) RawPacket {
	return buffer
}

func (p RawPacket) MetaPacketToRaw(packet *datatype.MetaPacket, tcpipChecksum bool) int {
	size := 0

	size += p.fillEthernet(packet, size)
	l3Offset := size

	switch packet.EthType {
	case layers.EthernetTypeIPv4:
		size += p.fillIPv4(packet, size)
	case layers.EthernetTypeARP:
		size += p.fillARP(packet, size)
	}

	if packet.EthType == layers.EthernetTypeIPv4 {
		switch packet.Protocol {
		case layers.IPProtocolICMPv4:
			size += p.fillICMPv4(packet, size)
		case layers.IPProtocolTCP:
			size += p.fillTCP(packet, size, l3Offset, tcpipChecksum)
		case layers.IPProtocolUDP:
			size += p.fillUDP(packet, size, l3Offset, tcpipChecksum)
		}
	}

	return size
}

const (
	ETHERNET_LEN = 14
	VLAN_LEN     = 4
)

func (p RawPacket) fillEthernet(packet *datatype.MetaPacket, start int) int {
	base := p[start:]
	offset := 0

	macIntToBytes(packet.MacDst, base[offset:])
	offset += MAC_ADDRESS_LEN
	macIntToBytes(packet.MacSrc, base[offset:])
	offset += MAC_ADDRESS_LEN

	if packet.Vlan != 0 {
		binary.BigEndian.PutUint16(base[offset:], uint16(layers.EthernetTypeDot1Q))
		offset += 2
		binary.BigEndian.PutUint16(base[offset:], uint16(packet.Vlan))
		offset += 2
	}

	binary.BigEndian.PutUint16(base[offset:], uint16(packet.EthType))
	return offset + 2
}

const (
	IPV4_VERSION_IHL_OFFSET     = 0
	IPV4_DSCP_ECN_OFFSET        = 1
	IPV4_TOTAL_LENGTH_OFFSET    = 2
	IPV4_ID_OFFSET              = 4
	IPV4_FLAGS_FRAGMENT_OFFSET  = 6
	IPV4_TTL_OFFSET             = 8
	IPV4_PROTOCOL_OFFSET        = 9
	IPV4_HEADER_CHECKSUM_OFFSET = 10
	IPV4_SIP_OFFSET             = 12
	IPV4_DIP_OFFSET             = 16
	IPV4_LEN                    = 20 // no options
)

const (
	IPV4_VERSION = 4
)

func (p RawPacket) fillIPv4(packet *datatype.MetaPacket, start int) int {
	base := p[start:]

	base[IPV4_VERSION_IHL_OFFSET] = (IPV4_VERSION << 4) | packet.IHL
	base[IPV4_DSCP_ECN_OFFSET] = 0
	binary.BigEndian.PutUint16(base[IPV4_TOTAL_LENGTH_OFFSET:], packet.PacketLen-uint16(start))
	binary.BigEndian.PutUint16(base[IPV4_ID_OFFSET:], packet.IpID)
	binary.BigEndian.PutUint16(base[IPV4_FLAGS_FRAGMENT_OFFSET:], packet.IpFlags)
	base[IPV4_TTL_OFFSET] = packet.TTL
	base[IPV4_PROTOCOL_OFFSET] = byte(packet.Protocol)
	binary.BigEndian.PutUint32(base[IPV4_SIP_OFFSET:], packet.IpSrc)
	binary.BigEndian.PutUint32(base[IPV4_DIP_OFFSET:], packet.IpDst)

	base[IPV4_HEADER_CHECKSUM_OFFSET] = 0
	base[IPV4_HEADER_CHECKSUM_OFFSET+1] = 0
	var csum uint32
	for i := 0; i < IPV4_LEN; i += 2 {
		csum += uint32(base[i]) << 8
		csum += uint32(base[i+1])
	}
	for csum > 0xFFFF {
		csum = (csum >> 16) + (csum & 0xFFFF)
	}
	binary.BigEndian.PutUint16(base[IPV4_HEADER_CHECKSUM_OFFSET:], ^uint16(csum))

	return IPV4_LEN
}

func (p RawPacket) fillARP(packet *datatype.MetaPacket, start int) int {
	return copy(p[start:], packet.RawHeader)
}

func (p RawPacket) fillICMPv4(packet *datatype.MetaPacket, start int) int {
	return copy(p[start:], packet.RawHeader)
}

const (
	TCP_SPORT_OFFSET       = 0
	TCP_DPORT_OFFSET       = 2
	TCP_SEQ_NUMBER_OFFSET  = 4
	TCP_ACK_NUMBER_OFFSET  = 8
	TCP_DATA_OFFSET_OFFSET = 12
	TCP_FLAGS_OFFSET       = 13 // NS not included
	TCP_WINDOW_SIZE_OFFSET = 14
	TCP_CHECKSUM_OFFSET    = 16
	TCP_URG_PTR_OFFSET     = 18
	TCP_OPTIONS_OFFSET     = 20
	TCP_MAX_OPTIONS_LEN    = 40
	TCP_MIN_LEN            = 20
)

const (
	TCP_OPTION_KIND_MSS_LEN            = 4
	TCP_OPTION_KIND_WINDOW_SCALE_LEN   = 3
	TCP_OPTION_KIND_SACK_PERMITTED_LEN = 2
)

func (p RawPacket) fillTCP(packet *datatype.MetaPacket, start, ipv4Offset int, checksum bool) int {
	if packet.Invalid || packet.TcpData == nil {
		return 0
	}

	base := p[start:]

	binary.BigEndian.PutUint16(base[TCP_SPORT_OFFSET:], packet.PortSrc)
	binary.BigEndian.PutUint16(base[TCP_DPORT_OFFSET:], packet.PortDst)

	binary.BigEndian.PutUint32(base[TCP_SEQ_NUMBER_OFFSET:], packet.TcpData.Seq)
	binary.BigEndian.PutUint32(base[TCP_ACK_NUMBER_OFFSET:], packet.TcpData.Ack)
	base[TCP_DATA_OFFSET_OFFSET] = packet.TcpData.DataOffset << 4
	base[TCP_FLAGS_OFFSET] = packet.TcpData.Flags
	binary.BigEndian.PutUint16(base[TCP_WINDOW_SIZE_OFFSET:], packet.TcpData.WinSize)
	binary.BigEndian.PutUint16(base[TCP_URG_PTR_OFFSET:], 0)

	optOffset := 0
	if packet.TcpData.MSS > 0 {
		base[TCP_OPTIONS_OFFSET+optOffset] = byte(layers.TCPOptionKindMSS)
		base[TCP_OPTIONS_OFFSET+optOffset+1] = TCP_OPTION_KIND_MSS_LEN
		binary.BigEndian.PutUint16(base[TCP_OPTIONS_OFFSET+optOffset+2:], packet.TcpData.MSS)
		optOffset += TCP_OPTION_KIND_MSS_LEN
	}
	if packet.TcpData.WinScale > 0 {
		base[TCP_OPTIONS_OFFSET+optOffset] = byte(layers.TCPOptionKindWindowScale)
		base[TCP_OPTIONS_OFFSET+optOffset+1] = TCP_OPTION_KIND_WINDOW_SCALE_LEN
		base[TCP_OPTIONS_OFFSET+optOffset+2] = packet.TcpData.WinScale
		optOffset += TCP_OPTION_KIND_WINDOW_SCALE_LEN
	}
	if packet.TcpData.SACKPermitted {
		base[TCP_OPTIONS_OFFSET+optOffset] = byte(layers.TCPOptionKindSACKPermitted)
		base[TCP_OPTIONS_OFFSET+optOffset+1] = TCP_OPTION_KIND_SACK_PERMITTED_LEN
		optOffset += TCP_OPTION_KIND_SACK_PERMITTED_LEN
	}
	if sackLen := len(packet.TcpData.Sack); sackLen > 0 {
		if sackLen&0x7 != 0 || sackLen > 32 { // not multiple of 8
			log.Debugf("SAck length %d incorrect", sackLen)
		} else {
			base[TCP_OPTIONS_OFFSET+optOffset] = byte(layers.TCPOptionKindSACK)
			base[TCP_OPTIONS_OFFSET+optOffset+1] = byte(sackLen + 2)
			copy(base[TCP_OPTIONS_OFFSET+optOffset+2:], packet.TcpData.Sack)
			optOffset += sackLen + 2
		}
	}
	length := int(packet.TcpData.DataOffset) << 2

	if checksum {
		binary.BigEndian.PutUint16(base[TCP_CHECKSUM_OFFSET:], p.tcpIPChecksum(layers.IPProtocolTCP, ipv4Offset, start, length))
	} else {
		binary.BigEndian.PutUint16(base[TCP_CHECKSUM_OFFSET:], 0)
	}

	return length
}

const (
	UDP_SPORT_OFFSET    = 0
	UDP_DPORT_OFFSET    = 2
	UDP_LENGTH_OFFSET   = 4
	UDP_CHECKSUM_OFFSET = 6
	UDP_LEN             = 8
)

func (p RawPacket) fillUDP(packet *datatype.MetaPacket, start, ipv4Offset int, checksum bool) int {
	base := p[start:]

	binary.BigEndian.PutUint16(base[UDP_SPORT_OFFSET:], packet.PortSrc)
	binary.BigEndian.PutUint16(base[UDP_DPORT_OFFSET:], packet.PortDst)
	binary.BigEndian.PutUint16(base[UDP_LENGTH_OFFSET:], UDP_LEN)
	if checksum {
		binary.BigEndian.PutUint16(base[UDP_CHECKSUM_OFFSET:], p.tcpIPChecksum(layers.IPProtocolUDP, ipv4Offset, start, UDP_LEN))
	} else {
		binary.BigEndian.PutUint16(base[UDP_CHECKSUM_OFFSET:], 0)
	}

	return UDP_LEN
}

func (p RawPacket) tcpIPChecksum(protocol layers.IPProtocol, ipv4Offset, tcpIPOffset int, length int) uint16 {
	csum := uint32(0)

	// pseudo header
	ipv4Layer := p[ipv4Offset:]
	csum += (uint32(ipv4Layer[IPV4_SIP_OFFSET]) + uint32(ipv4Layer[IPV4_SIP_OFFSET+2])) << 8
	csum += uint32(ipv4Layer[IPV4_SIP_OFFSET+1]) + uint32(ipv4Layer[IPV4_SIP_OFFSET+3])
	csum += (uint32(ipv4Layer[IPV4_DIP_OFFSET]) + uint32(ipv4Layer[IPV4_DIP_OFFSET+2])) << 8
	csum += uint32(ipv4Layer[IPV4_DIP_OFFSET+1]) + uint32(ipv4Layer[IPV4_DIP_OFFSET+3])
	csum += uint32(protocol) + uint32(length)
	// tcp/ip header
	tcpIPLayer := p[tcpIPOffset:]
	for i := 0; i < length; i += 2 {
		csum += uint32(binary.BigEndian.Uint16(tcpIPLayer[i : i+2]))
	}
	for csum > 0xFFFF {
		csum = (csum >> 16) + (csum & 0xFFFF)
	}

	return ^uint16(csum)
}

func min(x, y int) int {
	if x > y {
		return y
	}
	return x
}

func macIntToBytes(macInt datatype.MacInt, mac []byte) {
	binary.BigEndian.PutUint16(mac, uint16(macInt>>32))
	binary.BigEndian.PutUint32(mac[2:], uint32(macInt))
}
