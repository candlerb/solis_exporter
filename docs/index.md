# Overview

## Passive prometheus exporter

The primary function of this code is to sniff the messages between the data
logger and the inverter, and turn these passively collected values into
prometheus exporter values which can be scraped.  This gives a local source
of data with higher resolution than Solis Cloud (it's updated at 1 minute
intervals) and continues to work even if Solis Cloud is down.

## Modbus message injection

The secondary (optional) feature is rather more dubious: you can enable a
modbus TCP gateway which allows you to *inject* modbus messages onto the
RS485 bus to the inverter.

Modbus does *not* support multi-master operation, and the data logger is
unaware this is going on, so it's quite possible you will stomp on the data
logger's own messages.  The code here takes some care to check that the line
is idle before transmitting, but the data logger almost certainly doesn't.

Modbus messages are protected by a CRC, so the worst that should happen is
that occasionally a message fails, in which case the master should retry;
but it's possible that Bad Things™ will happen.

I added this functionality because I needed a remote control to set timed
charge/discharge on the inverter (which sits in my attic).  Since I'm only
going to do this once or twice a day, I'm prepared to accept the risk of an
occasional corrupted message on the modbus link.  It has worked fine for me
so far.

A safer way to do this would be to build a modbus proxy with two RS485
interfaces, and sit it in between the data logger and the inverter (acting
as a slave to the logger, and a master to the inverter).

Alternatively you could disconnect the data logger, so that your computer is
acting as the only modbus master, but then you'd lose the Solis Cloud
logging.  It would also be up to you to send the periodic messages to poll
the inverter.

I chose not to do this, because I wanted to keep the Solis Cloud monitoring
running (it has a nice app, and it can be used for support purposes by my PV
supplier).

## Security warning

If you enable the modbus TCP gateway, make sure it's bound to 127.0.0.1, and
if you expose it on the network, use firewall rules.  You definitely do not
want random unauthorized sources updating the parameters of your inverter.

!!! danger "SERIOUS DANGER"
    The inverter is a high-powered piece of machinery. By writing the
    configuration registers it's possible for it to be put into a state
    which does not comply with local supply regulations, or which is
    genuinely dangerous and may damage the inverter and/or other
    electrical equipment in your household.

    If you use the modbus TCP gateway functionality, configure it to limit
    accesses to known "safe" registers only.
