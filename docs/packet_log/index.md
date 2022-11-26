# Log of packet exchanges seen

These are a batch of packet exchanges captured between my Solis data logger and
RHI-6K-48ES-5G inverter.  I have broken them down into individual fields,
according to the
[Brian Coghlan document](https://www.scss.tcd.ie/Brian.Coghlan/Elios4you/RS485_MODBUS-Hybrid-BACoghlan-201811228-1854.pdf).

These are also included in `exporter_test.go` in the source.

## Once every minute

### Timings

The burst of readings taken every minute take about 5 seconds to complete.
Here is a typical example:

```
:30.822875 ->01040BB7000183C8
:30.909416 -<018402....
:31.831907 ->010480E80001983E
:31.911905 -<010402....
:31.951805 ->010481E20025B9DB
:32.108857 -<01044A....
:32.171728 ->010480E800299820
:32.299696 -<010452....
:32.832490 ->01048119002409EA
:32.988331 -<010448....
:33.522103 ->010481430005E9E1
:33.619940 -<01040A....
:34.151697 ->0104814C0016982F
:34.257655 -<01042C....
:34.792373 ->0104816600183823
:34.868770 -<010430....
:35.392103 ->01048189001409D3
:35.452987 -<010428....
:35.481984 ->010481DB0004A9CE
:35.519004 -<010408....
:35.552029 ->0103A8010001F5AA
:35.611887 -<010302....
:35.671904 ->010481E20025B9DB
:35.790824 -<01044A....
```

Then in the idle time there are two additional polls, about 30 seconds
apart:

```
:49.812598 ->010480E80001983E
:49.911437 -<01040231056CA3
```

```
:20.814119 ->010480E80001983E
:20.911090 -<01040231056CA3
```

### Read input registers: 2999

```
->01040BB7000183C8
-<018402C2C1
```

Failed: exception code 2 (illegal data address).

I suspect the logger also works with a different model of inverter, so it
probes this register to see which one it's talking to.

### Read input registers 33000: Product model

```
->010480E80001983E
-<01040231056CA3
```

Reg | Data | Description
----|------|------------
33000 | 3105 | Product model (hex)
CRC | 5CA3 |

<br />
Now it knows which model it's talking to.

I see two additional instances of this message sent every minute, making a
total of 3 per minute.  I think this is used as some sort of keepalive. 
Unfortunately, the exact timings are not consistent - so you can't predict
*exactly* when the line will be clear to inject messages.

### Read input registers 33250-33286: Meter data

```
->010481E20025B9DB
-<01044A0002095D00EA0000000000000000FFFFFE620000000000000000FFFFFE6200000121000000000000000000000121000001F90000000000000000000001F9FFAF1387000032880001BAD409A5
```

Reg | Data | Description
----|------|------------
33250 | 0002 | Meter placement (bitmap: 01=House side, 02=Grid side)
33251 | 095D | Meter AC voltage A (0.1V)
33252 | 00EA | Meter AC current A (0.01A)
33253 | 0000 | Meter AC voltage B (0.1V)
33254 | 0000 | Meter AC current B (0.01A)
33255 | 0000 | Meter AC voltage C (0.1V)
33256 | 0000 | Meter AC current C (0.01A)
33257+ | FFFFFE62 | Meter active power A (1W)
33259+ | 00000000 | Meter active power B (1W)
33261+ | 00000000 | Meter active power C (1W)
33263+ | FFFFFE62 | Meter total active power (1W)
33265+ | 00000121 | Meter reactive power A (1Var)
33267+ | 00000000 | Meter reactive power B (1Var)
33269+ | 00000000 | Meter reactive power C (1Var)
33271+ | 00000121 | Meter total reactive power (1Var)
33273+ | 000001F9 | Meter apparent power A (1VA)
33275+ | 00000000 | Meter apparent power B (1VA)
33277+ | 00000000 | Meter apparent power C (1VA)
33279+ | 000001F9 | Meter total apparent power (1VA)
33281 | FFAF | Meter power factor (?? x100)
33282 | 1387 | Meter grid frequency (0.01Hz)
33283+ | 00003288 | Meter total active energy imported from grid (0.01kWh)
33285+ | 0001BAD4 | Meter total active energy exported to grid (0.01kWh)
CRC | 09A5 |

<br />

### Read input registers 33000-33040: Product information and total power generation

```
->010480E800299820
-<01045231050032003C00013630333130353x3x3x3x3x3x3x3x3x3x00000000000000000000000000000000000000000016000B000D001300240020000000000CF5000000460000016D0029001B00000CF500000000xxxx
```

Reg | Data | Description
----|------|------------
33000 | 3105 | Product model (hex)
33001 | 0032 | DSP software version (hex)
33002 | 003C | LCD software version (hex)
33003 | 0001 | Protocol software version (hex)
33004+ | 36303331303.... | ASCII Serial number, nul padded
33020+ | 00000000 | Reserved
33022+ | 0016000B000D001300240020 | system time: yy mm dd HH MM SS
33028 | 0000 | Reserved
33029+ | 00000CF5 | Inverter total generation power (1kWh)
33031+ | 00000046 | Inverter power generation in the month (1kWh)
33033+ | 0000016D | Inverter last month's power generation (1kWh)
33035 | 0029 | Inverter power generation today (0.1kWh)
33036 | 001B | Inverter power generation yesterday (0.1kWh)
33037+ | 00000CF5 | Inverter power generation this year (1kWh)
33039+ | 00000000 | Inverter power generation last year (1kWh)
CRC | xxxx |

<br />

### Read input registers 33049-33084: Voltage and current data

```
->01048119002409EA
-<010448000E0001000F00000000000000000000000000000000000000000000000000000000000000000000003100000F3A0000095800000000001400000000FFFFFF9C0009FFF60000000A7C94
```

Reg | Data | Description
----|------|------------
33049 | 000E | DC voltage 1 (0.1V)
33050 | 0001 | DC current 1 (0.1A)
33051 | 000F | DC voltage 2 (0.1V)
33052 | 0000 | DC current 2 (0.1A)
33053 | 0000 | DC voltage 3 (0.1V)
33054 | 0000 | DC current 3 (0.1A)
33055 | 0000 | DC voltage 4 (0.1V)
33056 | 0000 | DC current 4 (0.1A)
33057+ | 00000000 | Total DC output power (1W)
33059+ | 0000.... | Reserved
33069 | 0031 | Reserved ??
33070 | 0000 | Reserved
33071 | 0F3A | DC bus voltage (0.1V)
33072 | 0000 | DC bus half voltage (0.1V)
33073 | 0958 | AB line voltage / phase A voltage (0.1V)
33074 | 0000 | BC line voltage / phase B voltage (0.1V)
33075 | 0000 | CA line voltage / phase C voltage (0.1V)
33076 | 0014 | phase A current (0.1A)
33077 | 0000 | phase B current (0.1A)
33078 | 0000 | phase C current (0.1A)
33079+ | FFFFFF9C | active power (1W)
33081+ | 0009FFF6 | reactive power (1Var)
33083+ | 0000000A | apparent power (1VA)
CRC | 7C94 |

<br />

### Read input registers 33091-33095: Operational parameters

```
->010481430005E9E1
-<01040A0000003500EF13880003A506
```

Reg | Data | Description
----|------|------------
33091 | 0000 | Standard working mode (enumerated, 0 = No response mode)
33092 | 0035 | National standard (Appendix III)
33093 | 00EF | Inverter temperature (0.1°C)
33094 | 1388 | Grid frequency (0.01Hz)
33095 | 0003 | Current state of the inverter (appendix II) 3 = Generating
CRC | A506 |

<br />

### Read input registers 33100-33121: Power and fault information

```
->0104814C0016982F
-<01042C00000000000000002AF803E80000000000000000000000000000000000000002000000000000000000000701F38E
```

Reg | Data | Description
----|------|------------
33100+ | 00000000 | limited active power adjustment rated power output value (1W)
33102+ | 00000000 | reactive power regulation rated power output value (1Var)
33104 | 2AF8 | Actual power limit (0.01%)
33105 | 03E8 | Actual power factor adjustment value (0.001)
33106 | 0000 | Reactive power (%, only for mode 4)
33107+ | 0000.... | reserved
33115 | 0002 | "set the flag bit" (appendix VIII)
33116 | 0000 | fault code 01 (appendix V)
33117 | 0000 | fault code 02
33118 | 0000 | fault code 03
33119 | 0000 | fault code 04
33120 | 0000 | fault code 05
33121 | 0701 | working status (appendix VI)
CRC | F38E |

<br />

### Read input registers 33126-33149: Meter power and battery state

```
->0104816600183823
-<01043000134598095D00EEFFFFFE55002301EA001C00010D2209580014001400631303000602E402E400000000012E00000000B2F6
```

Reg | Data | Description
----|------|------------
33126+ | 00134598 | Meter total active power generation (1Wh)
33128 | 095D | Meter voltage (0.1V)
33129 | 00EE | Meter current (0.1A)
33130+ | FFFFFE55 | Meter active power (1W, + = export, - = import)
33132 | 0023 | Energy storage control switch (Appendix VII) corresponds to write 43110 ?
33133 | 01EA | Battery voltage (0.1V)
33134 | 001C | Battery current (0.1A, + = charge, - = discharge?)
33135 | 0001 | Battery current direction (1 = discharge?)
33136 | 0D22 | LLCbus voltage (0.1V)
33137 | 0958 | Bypass AC voltage (0.1V)
33138 | 0014 | Bypass AC current (I think this is 0.01A)
33139 | 0014 | Battery capacity SOC %
33140 | 0063 | Battery health SOH %
33141 | 1303 | BMS battery voltage (0.01V)
33142 | 0006 | BMS battery current (0.01A)
33143 | 02E4 | BMS battery charge current limit (0.1A)
33144 | 02E4 | BMS battery discharge current limit (0.1A)
33145+ | 00000000 | BMS battery failure information
33147 | 012E | House load power (1W)
33148 | 0000 | Bypass load power (1W)
33149+ | 0000???? | Battery power (1W)
CRC | B2F6 |

<br />

NOTE: the battery power is a signed 32 bit value in registers 33149-33150,
but the logger only reads half of it (33149 only).  Therefore it cannot be
turned into a metric.

However, separate tests on the inverter show that the value read from
33149-33150 is simply the product of the battery voltage and current from
33133 and 33134, so this can be calculated externally.

### Read input registers 33161-33180: Battery charge and grid power totals

```
->01048189001409D3
-<010428000003E0001C00150000047A0022003300000081003000120000046D0000000000000986005E004BC806
```

Reg | Data | Description
----|------|------------
33161+ | 000003E0 | Total battery charge kWh
33163 | 001C | Battery charge today (0.1kWh)
33164 | 0015 | Battery charge yesterday (0.1kWh)
33165+ | 0000047A | Total battery discharge (1kWh)
33167 | 0022 | Battery discharge capacity (0.1kWh)
33168 | 0033 | Battery discharge power yesterday (0.1kWh)
33169+ | 00000081 | Total power imported from grid (1kWh)
33171 | 0030 | Grid power imported today (0.1kWh)
33172 | 0012 | Grid power imported yesterday (0.1kWh)
33173+ | 0000046D | Total power exported to grid(1kWh)
33175 | 0000 | Power exported to grid today (0.1kWh)
33176 | 0000 | Power exported to grid yesterday (0.1kWh)
33177+ | 00000986 | Total house load - consumption? (1kWh)
33179 | 005E | House load today (0.1kWh)
33180 | 004B | House load yesterday (0.1kWh)
CRC | C806 |

<br />

### Read input registers 33243-33246: Unknown/reserved

```
->010481DB0004A9CE
-<0104080000000000000000240D
```

Reg | Data | Description
----|------|------------
33243+ | 0000.... | Unknown/reserved (reads as zeros)
CRC | 240D |

<br />

### Read holding registers: 33250

```
->0103A8010001F5AA
-<01030200017984
```

Reg | Data | Description
----|------|------------
33250 | 0001 | Meter placement (bitmap: 01=House side, 02=Grid side)
CRC | 7984 |

<br />

Note that this register has been read before, but this time it is read by
itself, and using function code 03 instead of 04.

### REPEAT: Read input registers 33250-33286

```
->010481E20025B9DB
-<01044A....
```

For some reason, the meter data is read a second time, as part of the same
burst of communication.

## Once every 5 minutes

### Read input registers 33022-33027: System time

```
->010480FE00063838
-<01040C0016000B000D00130027001705C9
```

Reg | Data | Description
----|------|------------
33022+ | 0016000B000D001300270017 | System time: yy mm dd HH MM SS
CRC | 05C9 |

<br />

### Write multiple registers 43000-43005: Set real-time clock

```
->0110A7F800060C0016000B000D001300270018533C
-<0110A7F80006E28E
```

Reg | Data | Description
----|------|------------
43000+ | 0016000B000D001300270018 | yy mm dd HH MM SS
CRC | 533C |

<br />

Presumably setting the system clock from the Internet-obtained time.
