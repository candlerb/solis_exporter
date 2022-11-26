# References and related projects

## Solis operating modes

Timed charge/discharge can be configured manually via the front-panel
controls on the inverter.

See the [inverter manual](https://www.ginlong.com/rhi_inverter1/1952.html)
page 35 (PDF 19) for how to get to Advanced Settings, and pages 73-75 (PDF
38-39) for an explanation of the different operating modes.

[This video](https://www.youtube.com/watch?v=qCtriOPoS_Y) demonstrates the
process of configuring timed charge/discharge.

!!! warning
    The Solis manual says of the Advanced Settings:

    *This function is for authorised technicians only. Improper access and
    operation may result in abnormal results and damage to the inverter*

## Solis modbus registers

I found [this
thread](https://community.home-assistant.io/t/solis-inverter-modbus-integration/292553)
on the home-assistant forum extremely helpful, especially [post
31](https://community.home-assistant.io/t/solis-inverter-modbus-integration/292553/31)
which shows how to set the timed charge/discharge, and [post
54](https://community.home-assistant.io/t/solis-inverter-modbus-integration/292553/54)
which is for my exact model of inverter.

The same thread took me to the crucial
[PDF document](https://www.scss.tcd.ie/Brian.Coghlan/Elios4you/RS485_MODBUS-Hybrid-BACoghlan-201811228-1854.pdf)
containing the register layout.

To enable or disable timed charge/discharge, write to register 43310: 35 for
"run", 33 for "stop".  This must be written with the "Write Single Register"
modbus operation (function code 06).

```python
write_register(43110, 35, functioncode=6)
```

To set the time periods, write a block of 8 registers starting at 43143, in
a single "Write Multiple Registers" operation.  e.g.  charge 03:00 to 05:30,
no discharge:

```python
write_registers(43143, [3, 0, 5, 30, 0, 0, 0, 0])
```

To read these back, you need function code 03 ("read holding registers").  A
read with function code 04 ("read input registers") will return garbage.

Whilst the control panel shows multiple time periods for charge and
discharge, the above command only updates the first period.  It's possible
that subsequent registers are for the other slots, although the
documentation says these are "reserved".

See [packet log](../packet_log/) for packets captured between the data logger
and the inverter.

## modbus and serial specifications

* [modbus on Wikipedia](https://en.wikipedia.org/wiki/Modbus)
* [modbus.org specs](https://modbus.org/specs.php), in particular:
    * "MODBUS Protocol Specification"
    * "MODBUS Serial Line Protocol and Implementation Guide" - includes the
      character framing and timing requirements
* [kernel RS485 driver](https://www.kernel.org/doc/html/latest/driver-api/serial/serial-rs485.html)
    (not being used here, but interesting reference)
* [termios VMIN and VTIME](http://unixwiz.net/techtips/termios-vmin-vtime.html)
    (helpful to understand pyserial's `inter_byte_timeout` feature, and how
     go-serial-bugst uses VMIN)

Modbus over RS485 ("Modbus-RTU") is a truly awful protocol.  There is no
defined frame length: you are supposed to wait for a 3-character gap to
determine the end of the frame.  There is also no official way to determine
the difference between a command frame and a response frame, although there
are some heurstics (e.g.  function code 8, read registers, usually has an
even number of bytes in the command and an odd number of bytes in the
response).

For robustness, I decided to make this application decode each request and
response, and use knowledge of each message type to determine the number of
bytes to receive.  This is because the gap between the end of a request
packet and the start of the response may be as small as 3.5 character times
(less than 4ms at 9600bps)

The modbus serial spec says that the serial character framing *must* use 2
stop bits.  However, the default used by minimalmodbus is 1 stop bit, and it
seems to work fine.  Ideally I'd like to hook some sort of protocol analyzer
onto the connection to see whether the data logger and the inverter are
sending with 1 stop bit or 2.

## esphome-externalcomponents

[grob6000/esphome-externalcomponents](https://github.com/grob6000/esphome-externalcomponents#solis_s5)
is another project which does passive sniffing of messages between the
inverter and logger.  This gave me my first clue that it was possible to
[piggyback](https://github.com/grob6000/esphome-externalcomponents/blob/master/solis_piggyback_schematic_0.pdf)
onto the existing Wifi dongle.  The project describes an ESP32-based
collector which is small enough to fit *inside* the Solis data logger
itself.

However, it doesn't export to prometheus metrics: it provides the ESPHome
[native API](https://esphome.io/components/api.html).  It also
(intentionally!) doesn't allow injection of modbus messages.  The circuit
doesn't even connect to the UART TX pin.

## modbus-sniffer

My first testing used
[modbus-sniffer](https://github.com/alerighi/modbus-sniffer) as a quick way
to check that I could capture the messages and decode them by hand.  It
doesn't decode modbus message bodies, but just waits for a pause in the
transmission to detect the end of a packet.  It does validate the CRC
though.

I found that the default timeout of 1500µs was too low to work reliably,
but it was fine when increased to 2000µs:

```bash
./sniffer -p /dev/ttyUSB0 -l --interval 2000 -o sniff.pcap
```

It creates pcap files, but `tcpdump` refuses to read them (unknown
protocol).  `tshark` can, but it doesn't show the contents.

## minimalmodbus

[minimalmodbus](https://github.com/pyhys/minimalmodbus) is a small python
library for acting as a modbus master.  I used it for my first successful
attempt to set a timed charge period.

It doesn't do the passive sniffing of other masters on the bus, but you can
do a quick-and-dirty message exchange if you wait for the line to be idle
for, say, 1.5 seconds (and this usually works):

```python
import minimalmodbus
i = minimalmodbus.Instrument(port="/dev/ttyUSB0", slaveaddress=1, debug=True)
i.serial.baudrate=9600   # oddly they default to 19200
def ready(i):
    i.serial.timeout=1.5
    i.serial.reset_input_buffer()
    i.serial.read(100000)
    i.serial.timeout=0.2
    return i

print(ready(i).read_registers(43143, 8))
```

It doesn't support TCP though, so you can't use it to talk to the gateway
feature of solis_exporter.

## umodbus

[umodbus](https://pypi.org/project/uModbus/) is another python modbus
library, capable of driving either RTU or TCP connections.  It's used by
[favalex/modbus-cli](https://github.com/favalex/modbus-cli).  I haven't
tried either of these.

## pymodbus

[pymodbus](https://pymodbus.readthedocs.io/en/latest/) is another modbus
client/server library with RTU and TCP support.  Its CLI is in the form of a
[REPL](https://github.com/riptideio/pymodbus/blob/dev/pymodbus/repl/client/README.md)
evaluator.

## mbpoll

[mbpoll](https://github.com/epsilonrt/mbpoll) is built on libmodbus and is
available in the standard Ubuntu package repositories
(`apt install mbpoll`).  It can use either RTU or TCP.

You can use it to inject TCP messages into the gateway, but I cannot
recommend it because of what I consider a major design flaw: it subtracts
one from the supplied register address unless you remember the `-0` flag. 
As a result, it's *really* easy to read and write the wrong registers by
accident, which could be catastrophic.

```sh
# CORRECT EXAMPLES FOR TIMED CHARGE/DISCHARGE
mbpoll 127.0.0.1 -p 1502 -0 -1 -o 10 -r 43110                   # read operating mode
mbpoll 127.0.0.1 -p 1502 -0 -1 -o 10 -r 43110 35                # set operating mode
mbpoll 127.0.0.1 -p 1502 -0 -1 -o 10 -r 43143 -c 8              # read times
mbpoll 127.0.0.1 -p 1502 -0 -1 -o 10 -r 43143 3 0 5 30 0 0 0 0  # set times
```

* `-p 1502`: TCP port
* `-0`: use zero register offset!!
* `-1`: run only once (not repeating in a loop!)
* `-o 10`: timeout 10 seconds
* `-r 43143`: register base
* `-c 8`: register count (for reading multiple registers)

Furthermore: read requests are sent by default using function code 3.  This
is also the case if you supply flag `-t 4`.  To get function code 4, you
have to set `-t 3`.  Go figure!!

## mbusd

[mbusd](https://github.com/3cky/mbusd) is a modbus TCP to RTU (RS485)
gateway, written in C.  It doesn't include passive sniffing of requests
generated by other masters - which is to be expected, since modbus does not
officially allow multiple masters on the same bus anyway.
