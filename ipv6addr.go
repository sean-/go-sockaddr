package sockaddr

import (
	"fmt"
	"math/big"
	"net"
)

type (
	// IPv6Address is a named type representing an IPv6 address.
	IPv6Address *big.Int

	// IPv6Network is a named type representing an IPv6 network.
	IPv6Network *big.Int

	// IPv6Mask is a named type representing an IPv6 network mask.
	IPv6Mask *big.Int
)

// IPv6HostPrefix is a constant represents a /128 IPv6 Prefix.
const IPv6HostPrefix = IPPrefixLen(128)

// ipv6HostMask is an unexported big.Int representing a /128 IPv6 address.
// This value must be a constant and always set to all ones.
var ipv6HostMask IPv6Mask

func init() {
	biMask := new(big.Int)
	biMask.SetBytes([]byte{
		0xff, 0xff,
		0xff, 0xff,
		0xff, 0xff,
		0xff, 0xff,
		0xff, 0xff,
		0xff, 0xff,
		0xff, 0xff,
		0xff, 0xff,
	},
	)
	ipv6HostMask = IPv6Mask(biMask)
}

// IPv6Addr implements a convenience wrapper around the union of Go's
// built-in net.IP and net.IPNet types.  In UNIX-speak, IPv6Addr implements
// `sockaddr` when the the address family is set to AF_INET6
// (i.e. `sockaddr_in6`).
type IPv6Addr struct {
	IPAddr
	Address IPv6Address
	Mask    IPv6Mask
	Port    IPPort
}

// NewIPv6Addr creates an IPv6Addr from a string.  String can be in the form of
// an an IPv6:port (e.g. `[2001:4860:0:2001::68]:80`, in which case the mask is
// assumed to be a /128), an IPv6 address (e.g. `2001:4860:0:2001::68`, also
// with a `/128` mask), an IPv6 CIDR (e.g. `2001:4860:0:2001::68/64`, which has
// its IP port initialized to zero).  ipv6Str can not be a hostname.
//
// NOTE: Many net.*() routines will initialize and return an IPv4 address.
// Always test to make sure the address returned cannot be converted to a 4 byte
// array using To4().
func NewIPv6Addr(ipv6Str string) (IPv6Addr, error) {
	v6Addr := false
LOOP:
	for i := 0; i < len(ipv6Str); i++ {
		switch ipv6Str[i] {
		case '.':
			break LOOP
		case ':':
			v6Addr = true
			break LOOP
		}
	}

	if !v6Addr {
		return IPv6Addr{}, fmt.Errorf("Unable to resolve %+q as an IPv6 address, appears to be an IPv4 address", ipv6Str)
	}

	// Attempt to parse ipv6Str as a /128 host with a port number.
	tcpAddr, err := net.ResolveTCPAddr("tcp6", ipv6Str)
	if err == nil {
		ipv6 := tcpAddr.IP.To16()
		if ipv6 == nil {
			return IPv6Addr{}, fmt.Errorf("Unable to resolve %+q as a 16byte IPv6 address", ipv6Str)
		}

		ipv6BigIntAddr := new(big.Int)
		ipv6BigIntAddr.SetBytes(ipv6)

		ipv6BigIntMask := new(big.Int)
		ipv6BigIntMask.Set(ipv6HostMask)

		ipv6Addr := IPv6Addr{
			Address: IPv6Address(ipv6BigIntAddr),
			Mask:    IPv6Mask(ipv6BigIntMask),
			Port:    IPPort(tcpAddr.Port),
		}

		return ipv6Addr, nil
	}

	// Parse as a naked IPv6 address.  Trim square brackets if present.
	if len(ipv6Str) > 2 && ipv6Str[0] == '[' && ipv6Str[len(ipv6Str)-1] == ']' {
		ipv6Str = ipv6Str[1 : len(ipv6Str)-1]
	}
	ip := net.ParseIP(ipv6Str)
	if ip != nil {
		ipv6 := ip.To16()
		if ipv6 == nil {
			return IPv6Addr{}, fmt.Errorf("Unable to string convert %+q to a 16byte IPv6 address", ipv6Str)
		}

		ipv6BigIntAddr := new(big.Int)
		ipv6BigIntAddr.SetBytes(ipv6)

		ipv6BigIntMask := new(big.Int)
		ipv6BigIntMask.Set(ipv6HostMask)

		ipv6Addr := IPv6Addr{
			Address: IPv6Address(ipv6BigIntAddr),
			Mask:    IPv6Mask(ipv6BigIntMask),
		}

		return ipv6Addr, nil
	}

	// Parse as an IPv6 CIDR
	ipAddr, network, err := net.ParseCIDR(ipv6Str)
	if err == nil {
		ipv6 := ipAddr.To16()
		if ipv6 == nil {
			return IPv6Addr{}, fmt.Errorf("Unable to convert %+q to a 16byte IPv6 address", ipv6Str)
		}

		ipv6BigIntAddr := new(big.Int)
		ipv6BigIntAddr.SetBytes(ipv6)

		ipv6BigIntMask := new(big.Int)
		ipv6BigIntMask.SetBytes(network.Mask)

		ipv6Addr := IPv6Addr{
			Address: IPv6Address(ipv6BigIntAddr),
			Mask:    IPv6Mask(ipv6BigIntMask),
		}
		return ipv6Addr, nil
	}

	return IPv6Addr{}, fmt.Errorf("Unable to parse %+q to an IPv6 address: %v", ipv6Str, err)
}

// AddressBinString returns a string with the IPv6Addr's Address represented
// as a sequence of '0' and '1' characters.  This method is useful for
// debugging or by operators who want to inspect an address.
func (ipv6 IPv6Addr) AddressBinString() string {
	bi := big.Int(*ipv6.Address)
	return fmt.Sprintf("%0128s", bi.Text(2))
}

// AddressHexString returns a string with the IPv6Addr address represented as
// a sequence of hex characters.  This method is useful for debugging or by
// operators who want to inspect an address.
func (ipv6 IPv6Addr) AddressHexString() string {
	bi := big.Int(*ipv6.Address)
	return fmt.Sprintf("%032s", bi.Text(16))
}

// Cmp returns 0 if a SockAddr is equal to the receiving IPv6Addr, -1 if it
// should sort first, or 1 if it should sort after.
func (ipv6 IPv6Addr) CmpAddress(sa SockAddr) int {
	ipv6b, ok := sa.(IPv6Addr)
	if !ok {
		return SortOrderDifferentTypes
	}

	ipv6aBigInt := new(big.Int)
	ipv6aBigInt.Set(ipv6.Address)
	ipv6bBigInt := new(big.Int)
	ipv6bBigInt.Set(ipv6b.Address)

	return ipv6aBigInt.Cmp(ipv6bBigInt)
}

// CmpPort returns 0 if a SockAddr is equal to the receiving IPv6Addr, -1
// if it should sort first, or 1 if it should sort after.
func (ipv6 IPv6Addr) CmpPort(sa SockAddr) int {
	var saPort IPPort
	switch v := sa.(type) {
	case IPv4Addr:
		saPort = v.Port
	case IPv6Addr:
		saPort = v.Port
	default:
		return SortOrderDifferentTypes
	}

	switch {
	case ipv6.Port == saPort:
		return 0
	case ipv6.Port < saPort:
		return -1
	default:
		return 1
	}
}

// Contains returns true if the SockAddr is contained within the receiver.
func (ipv6 IPv6Addr) Contains(sa SockAddr) bool {
	ipv6b, ok := sa.(IPv6Addr)
	if !ok {
		return false
	}

	return ipv6.ContainsNetwork(ipv6b)
}

// ContainsNetwork returns true if the network from IPv6Addr is contained within
// the receiver.
func (x IPv6Addr) ContainsNetwork(y IPv6Addr) bool {
	{
		xIPv6 := x.FirstUsable().(IPv6Addr)
		yIPv6 := y.FirstUsable().(IPv6Addr)
		if xIPv6.CmpAddress(yIPv6) > 1 {
			return false
		}
	}

	{
		xIPv6 := x.LastUsable().(IPv6Addr)
		yIPv6 := y.LastUsable().(IPv6Addr)
		if xIPv6.CmpAddress(yIPv6) <= 0 {
			return false
		}
	}
	return true
}

// DialPacketArgs returns the arguments required to be passed to
// net.DialUDP().  If the Mask of ipv6 is not a /128 or the Port is 0,
// DialPacketArgs() will fail.  See Host() to create an IPv6Addr with its
// mask set to /128.
func (ipv6 IPv6Addr) DialPacketArgs() (network, dialArgs string) {
	ipv6Mask := big.Int(*ipv6.Mask)
	if ipv6Mask.Cmp(ipv6HostMask) != 0 || ipv6.Port == 0 {
		return "udp6", ""
	}
	return "udp6", fmt.Sprintf("[%s]:%d", ipv6.NetIP().String(), ipv6.Port)
}

// DialStreamArgs returns the arguments required to be passed to
// net.DialTCP().  If the Mask of ipv6 is not a /128 or the Port is 0,
// DialStreamArgs() will fail.  See Host() to create an IPv6Addr with its
// mask set to /128.
func (ipv6 IPv6Addr) DialStreamArgs() (network, dialArgs string) {
	ipv6Mask := big.Int(*ipv6.Mask)
	if ipv6Mask.Cmp(ipv6HostMask) != 0 || ipv6.Port == 0 {
		return "tcp6", ""
	}
	return "tcp6", fmt.Sprintf("[%s]:%d", ipv6.NetIP().String(), ipv6.Port)
}

// Equal returns true if a SockAddr is equal to the receiving IPv4Addr.
func (ipv6a IPv6Addr) Equal(sa SockAddr) bool {
	if sa.Type() != TypeIPv6 {
		return false
	}
	ipv6b, ok := sa.(IPv6Addr)
	if !ok {
		return false
	}

	// Now that the type conversion checks are complete, verify the data
	// is equal.
	if ipv6a.NetIP().String() != ipv6b.NetIP().String() {
		return false
	}

	if ipv6a.NetIPNet().String() != ipv6b.NetIPNet().String() {
		return false
	}

	if ipv6a.Port != ipv6b.Port {
		return false
	}

	return true
}

// FirstUsable returns an IPv6Addr set to the first address following the
// network prefix.  The first usable address in a network is normally the
// gateway and should not be used except by devices forwarding packets
// between two administratively distinct networks (i.e. a router).  This
// function does not discriminate against first usable vs "first address that
// should be used."  For example, FirstUsable() on "2001:0db8::0003/64" would
// return "2001:0db8::00011".
func (ipv6 IPv6Addr) FirstUsable() IPAddr {
	return IPv6Addr{
		Address: IPv6Address(ipv6.NetworkAddress()),
		Mask:    ipv6HostMask,
	}
}

// Host returns a copy of ipv6 with its mask set to /128 so that it can be
// used by DialPacketArgs(), DialStreamArgs(), ListenPacketArgs(), or
// ListenStreamArgs().
func (ipv6 IPv6Addr) Host() IPAddr {
	// Nothing should listen on a broadcast address.
	return IPv6Addr{
		Address: ipv6.Address,
		Mask:    ipv6HostMask,
		Port:    ipv6.Port,
	}
}

// IPPort returns the Port number as a uint16
func (ipv6 IPv6Addr) IPPort() IPPort {
	return ipv6.Port
}

// LastUsable returns the last address in a given network.
func (ipv6 IPv6Addr) LastUsable() IPAddr {
	addr := new(big.Int)
	addr.Set(ipv6.Address)

	mask := new(big.Int)
	mask.Set(ipv6.Mask)

	negMask := new(big.Int)
	negMask.Xor(ipv6HostMask, mask)

	lastAddr := new(big.Int)
	lastAddr.And(addr, mask)
	lastAddr.Or(lastAddr, negMask)

	return IPv6Addr{
		Address: IPv6Address(lastAddr),
		Mask:    ipv6HostMask,
	}
}

// ListenPacketArgs returns the arguments required to be passed to
// net.ListenUDP().  If the Mask of ipv6 is not a /128, ListenPacketArgs()
// will fail.  See Host() to create an IPv6Addr with its mask set to /128.
func (ipv6 IPv6Addr) ListenPacketArgs() (network, listenArgs string) {
	ipv6Mask := big.Int(*ipv6.Mask)
	if ipv6Mask.Cmp(ipv6HostMask) != 0 {
		return "udp6", ""
	}
	return "udp6", fmt.Sprintf("[%s]:%d", ipv6.NetIP().String(), ipv6.Port)
}

// ListenStreamArgs returns the arguments required to be passed to
// net.ListenTCP().  If the Mask of ipv6 is not a /128, ListenStreamArgs()
// will fail.  See Host() to create an IPv6Addr with its mask set to /128.
func (ipv6 IPv6Addr) ListenStreamArgs() (network, listenArgs string) {
	ipv6Mask := big.Int(*ipv6.Mask)
	if ipv6Mask.Cmp(ipv6HostMask) != 0 {
		return "tcp6", ""
	}
	return "tcp6", fmt.Sprintf("[%s]:%d", ipv6.NetIP().String(), ipv6.Port)
}

// Maskbits returns the number of network mask bits in a given IPv6Addr.  For
// example, the Maskbits() of "2001:0db8::0003/64" would return 64.
func (ipv6 IPv6Addr) Maskbits() int {
	maskOnes, _ := ipv6.NetIPNet().Mask.Size()

	return maskOnes
}

// NetIP returns the address as a net.IP.
func (ipv6 IPv6Addr) NetIP() *net.IP {
	x := make(net.IP, IPv6len)
	b := big.Int(*ipv6.Address)
	copy(x, b.Bytes())
	return &x
}

// NetIPMask create a new net.IPMask from the IPv6Addr.
func (ipv6 IPv6Addr) NetIPMask() *net.IPMask {
	ipv6Mask := make(net.IPMask, IPv6len)
	m := big.Int(*ipv6.Mask)
	copy(ipv6Mask, m.Bytes())
	return &ipv6Mask
}

// Network returns a pointer to the net.IPNet within IPv4Addr receiver.
func (ipv6 IPv6Addr) NetIPNet() *net.IPNet {
	ipv6net := &net.IPNet{}
	ipv6net.IP = make(net.IP, IPv6len)
	copy(ipv6net.IP, *ipv6.NetIP())
	ipv6net.Mask = *ipv6.NetIPMask()
	return ipv6net
}

// NetworkPrefix returns the network prefix or network address for a given
// network.
func (ipv6 IPv6Addr) Network() IPAddr {
	return IPv6Addr{
		Address: IPv6Address(ipv6.NetworkAddress()),
		Mask:    ipv6.Mask,
	}
}

// NetworkAddress returns an IPv6Network of the IPv6Addr's network address.
func (ipv6 IPv6Addr) NetworkAddress() IPv6Network {
	addr := new(big.Int)
	addr.SetBytes((*ipv6.Address).Bytes())

	mask := new(big.Int)
	mask.SetBytes(*ipv6.NetIPMask())

	netAddr := new(big.Int)
	netAddr.And(addr, mask)

	return IPv6Network(netAddr)
}

// Octets returns a slice of the 16 octets in an IPv6Addr's Address.  The
// order of the bytes is big endian.
func (ipv6 IPv6Addr) Octets() []int {
	octets := make([]int, IPv6len)
	b := big.Int(*ipv6.Address)
	for i, o := range b.Bytes() {
		octets[i] = int(o)
	}
	return octets
}

// SetPort is a setter method to set an IPv6Addr's port number
func (ipv6 IPv6Addr) SetPort(p uint16) IPAddr {
	// Operating on a copy of ipv6, return the mutated struct
	ipv6.Port = IPPort(p)
	return ipv6
}

// String returns a string representation of the IPv6Addr
func (ipv6 IPv6Addr) String() string {
	if ipv6.Port != 0 {
		return fmt.Sprintf("[%s]:%d", ipv6.NetIP().String(), ipv6.Port)
	}

	if ipv6.Maskbits() == 128 {
		return ipv6.NetIP().String()
	}

	return fmt.Sprintf("%s/%d", ipv6.NetIP().String(), ipv6.Maskbits())
}

// Type is used as a type switch and returns TypeIPv6
func (IPv6Addr) Type() SockAddrType {
	return TypeIPv6
}
