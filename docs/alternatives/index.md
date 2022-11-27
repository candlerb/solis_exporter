# Alternative ways of integrating with Solis inverters

I have not tested any of these, but I came across them in my initial
research.  The advantage of these is that no modification to the wifi data
logger is required.

## Binary logger updates

The wifi logger sends its data to Solis Cloud, but you can also configure a
second IP address to send messages to, and this protocol has been partially
reverse engineered.  This is configured via the data logger's web admin
interface.

* [ginlong-wifi](https://github.com/graham0/ginlong-wifi) - `nc -l` (bash)
  and python versions - 2G inverter, 206 byte frames
* [ginlong-mqtt](https://github.com/dpoulson/ginlong-mqtt) - python/mqtt -
  4G Mini inverter, 270 byte frames; includes a
  [protocol doc](https://github.com/dpoulson/ginlong-mqtt/blob/master/Protocol).

It could be that neither of these will work with a 5G inverter.

## Scraping of status.html from the data logger

The wifi data logger has its own configuration web interface, and a small
subset of inverter status data is available though it:

```
curl -u admin:admin http://x.x.x.x/status.html
```

* [solis-inverter](https://github.com/fss/solis-inverter) - node.js, serves a custom JSON payload.
  Found via [this post](https://community.openenergymonitor.org/t/working-integration-with-ginlong-solis-pv-inverter-wifi-stick/15357).
* [solismon3](https://github.com/NosIreland/solismon3) - python, includes a prometheus exporter

To decode the data from status.html, see
[solis_inverter_client.js](https://github.com/fss/solis-inverter/blob/master/lib/solis_inverter_client.js)

Available information through this route seems to be limited to
instantaneous power, total energy, and wifi status.

## Scraping of Solis Cloud (m.jinlong.com)

* [https://github.com/dkruyt/ginlong-scraper](ginlong-scraper) +
  [blog](https://blog.kruyt.org/ginlong-scraper/) - python, sends to influxdb and/or
  mqtt, has grafana dashboard for influxdb

You're limited to the resolution that Solis Cloud stores (5 minutes).

## Direct RS485 connection without data logger

I found some projects and references for making active queries via RS485
connected directly to the inverter, without any data logger.

* [solis2mqtt](https://github.com/incub77/solis2mqtt) - includes hardware and connector
  details
* [reddit thread](https://www.reddit.com/r/homeassistant/comments/usavoh/ginlong_solis_pv_inverter_to_mqtt_and_home/)
  which links to [markgdev python script](https://gist.github.com/markgdev/ce2dbf9002385cbe5a35b81985f9c84a)
  for collecting and pushing to mqtt
* [openenergymonitor thread](https://community.openenergymonitor.org/t/getting-data-from-inverters-via-an-rs485-connection/8377/26?page=2)
  which also includes some discussion of Raspberry Pi RS485 hardware
* [Brian Dorey's blog
  post](https://www.briandorey.com/post/solar-upgrade-solis-1-5kw-inverter-raspberry-pi-rs485-logging),
  although this is for a different inverter model to mine with a different
  register map.  It describes how you might need to use 10KΩ pull-up and
  pull-down resistors.

Using this approach you would connect your RS485 interface directly to the
inverter.  This requires an unusual Exceedconn EC04681-2023-BF connector but
you can find it if you look hard enough - e.g. 
[here](https://www.ebay.co.uk/itm/275517047561).  I read somewhere that this
type of connector is used on aircraft.

This approach avoids any risk of colliding messages on the bus, but you lose
the ability to use the Solis Cloud portal.

solis_exporter could be modified to work this way: it would just need to
inject its own polling messages periodically.
