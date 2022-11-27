# Metrics

## Sample metrics

Here are some examples of metrics returned:

```
go_build_info{checksum="",path="github.com/candlerb/solis_exporter",version="(devel)"} 1
promhttp_metric_handler_errors_total{cause="encoding"} 0
promhttp_metric_handler_errors_total{cause="gathering"} 0
solis_battery_current 15.100000000000001
solis_battery_soc 28
solis_battery_soh 99
solis_battery_voltage 50.6
solis_bms_battery_current 13.5
solis_bms_battery_voltage 50.06
solis_bms_charge_limit_current 74
solis_bms_discharge_limit_current 74
solis_bms_failure_flags{code="01"} 0
solis_bms_failure_flags{code="02"} 0
solis_grid_current{phase="U"} 1.45
solis_grid_current{phase="V"} 0
solis_grid_current{phase="W"} 0
solis_grid_energy{period="all",type="export"} 1150.2
solis_grid_energy{period="all",type="import"} 205.56
solis_grid_frequency 50.07
solis_grid_power_active 2
solis_grid_power_apparent 235
solis_grid_power_factor -0.81
solis_grid_power_reactive 235
solis_grid_voltage{phase="U"} 239.9
solis_grid_voltage{phase="V"} 0
solis_grid_voltage{phase="W"} 0
solis_inverter_ac_current{phase="U"} 2.1
solis_inverter_ac_current{phase="V"} 0
solis_inverter_ac_current{phase="W"} 0
solis_inverter_ac_voltage{phase="U"} 239.20000000000002
solis_inverter_ac_voltage{phase="V"} 0
solis_inverter_ac_voltage{phase="W"} 0
solis_inverter_backup_current 0.2
solis_inverter_backup_power 0
solis_inverter_backup_voltage 239.10000000000002
solis_inverter_dc_current{pv="1"} 1.8
solis_inverter_dc_current{pv="2"} 1.7000000000000002
solis_inverter_dc_power 849
solis_inverter_dc_voltage{pv="1"} 207.60000000000002
solis_inverter_dc_voltage{pv="2"} 280.1
solis_inverter_energy{period="all",type="charge"} 1065
solis_inverter_energy{period="all",type="discharge"} 1224
solis_inverter_energy{period="all",type="export"} 1150
solis_inverter_energy{period="all",type="import"} 205
solis_inverter_energy{period="all",type="load"} 2562
solis_inverter_energy{period="all",type="yield"} 3378
solis_inverter_energy{period="day",type="charge"} 1
solis_inverter_energy{period="day",type="discharge"} 1.4000000000000001
solis_inverter_energy{period="day",type="export"} 0
solis_inverter_energy{period="day",type="import"} 1.5
solis_inverter_energy{period="day",type="load"} 3.3000000000000003
solis_inverter_energy{period="day",type="yield"} 1.3
solis_inverter_energy{period="day-1",type="charge"} 6.2
solis_inverter_energy{period="day-1",type="discharge"} 8.9
solis_inverter_energy{period="day-1",type="export"} 1.2000000000000002
solis_inverter_energy{period="day-1",type="import"} 0
solis_inverter_energy{period="day-1",type="load"} 10.200000000000001
solis_inverter_energy{period="day-1",type="yield"} 9.1
solis_inverter_energy{period="month",type="yield"} 131
solis_inverter_energy{period="month-1",type="yield"} 365
solis_inverter_energy{period="year",type="yield"} 3378
solis_inverter_energy{period="year-1",type="yield"} 0
solis_inverter_fault_flags{code="01"} 0
solis_inverter_fault_flags{code="02"} 0
solis_inverter_fault_flags{code="03"} 0
solis_inverter_fault_flags{code="04"} 0
solis_inverter_fault_flags{code="05"} 0
solis_inverter_frequency 50.06
solis_inverter_info{dsp_version="0032",lcd_version="003C",model="3105",protocol_version="0001",serial="603105XXXXXXXXXX"} 1
solis_inverter_load_power 0
solis_inverter_operating_state 3
solis_inverter_power_active 70
solis_inverter_power_apparent 70
solis_inverter_power_reactive 0
solis_inverter_storage_control_flags 35
solis_inverter_temperature 20.700000000000003
solis_inverter_working_status_flags 1793
solis_serial_errors_total{error="crc_failed"} 0
solis_serial_errors_total{error="decode_failed"} 0
solis_serial_errors_total{error="response_mismatch"} 0
solis_serial_errors_total{error="timeout"} 0
solis_serial_last_message_time_seconds 1.6694584415372543e+09
solis_serial_messages_total{source="injected"} 2
solis_serial_messages_total{source="sniffed"} 230
```

## Units

I have chosen to return watts for power, rather than kilowatts.  This is to
be consistent with volts and amps.  However, accumulated energy use is in
conventional kWh units (rather than Wh or J)

I left out the units from the metric names; `battery_voltage_volts` would be
somewhat redundant.

## Correspondence with Solis Cloud stats

In Solis Cloud, under the "Inverter" details page, the graph has a "Select
Parameters" set of checkboxes.  Here I show how they correspond to the
exporter output.  In a few cases, these appear to be calculated from the
collected data; I have given the corresponding PromQL queries.

### Inverter

Solis Cloud parameter | solis_exporter metric
----------------------|----------------------
DC Voltage PV1 | `solis_inverter_dc_voltage{pv="1"}`
DC Voltage PV2 | `solis_inverter_dc_voltage{pv="2"}`
DC Current PV1 | `solis_inverter_dc_current{pv="1"}`
DC Current PV2 | `solis_inverter_dc_current{pv="2"}`
DC Power PV1 | `solis_inverter_dc_voltage{pv="1"} * solis_inverter_dc_current`
DC Power PV2 | `solis_inverter_dc_voltage{pv="2"} * solis_inverter_dc_current`
U_AC Voltage | `solis_inverter_ac_voltage{phase="U"}`
V_AC Voltage | `solis_inverter_ac_voltage{phase="V"}`
W_AC Voltage | `solis_inverter_ac_voltage{phase="W"}`
U_AC Current | `solis_inverter_ac_current{phase="U"}`
V_AC Current | `solis_inverter_ac_current{phase="V"}`
W_AC Current | `solis_inverter_ac_current{phase="W"}`
AC Output Frequency | `solis_inverter_frequency`
Total Power | `solis_inverter_dc_power`
Daily Yield | `solis_inverter_energy{period="day",type="yield"}`
Total Yield | `solis_inverter_energy{period="all",type="yield"}`
Inverter internal operating temperature | `solis_inverter_temperature`

<br />
Although "Total Power" is collected from the inverter, it appears that it is
simply the total DC power across all the strings, since you can get a
matching result using:

```
sum by (instance) (solis_inverter_dc_voltage * solis_inverter_dc_current)
```

There is a separate metric `solis_inverter_power_active` (collected from
modbus 33079-33080), but I am not entirely sure what it represents; it can
go negative.

### Grid

Solis Cloud parameter | solis_exporter metric
----------------------|----------------------
U_Grid Voltage | `solis_grid_voltage{phase="U"}`
V_Grid Voltage | `solis_grid_voltage{phase="V"}`
W_Grid Voltage | `solis_grid_voltage{phase="W"}`
U_Grid Current | `solis_grid_current{phase="U"}`
V_Grid Current | `solis_grid_current{phase="V"}`
W_Grid Current | `solis_grid_current{phase="W"}`
Grid Total Active Power | `solis_grid_power_active`
Grid Total Reactive Power | `solis_grid_power_reactive`
Grid Total Apparent Power | `solis_grid_power_apparent `
Grid Power Factor | `solis_grid_power_factor`
Grid Frequency | `solis_grid_frequency`
Daily Energy to Grid | `solis_inverter_energy{period="day",type="export"}`
Total Energy to Grid | `solis_inverter_energy{period="all",type="export"}`
Daily Energy from Grid | `solis_inverter_energy{period="day",type="import"}`
Total Energy from Grid | `solis_inverter_energy{period="all",type="import"}`

<br />
Note that the total energy to/from Grid is also available with higher
resolution (0.01kWh) from `solis_grid_energy{type=~"export|import"}`, which
comes from different fields in the modbus data but doesn't include a daily
figure.

### Battery

Solis Cloud parameter | solis_exporter metric
----------------------|----------------------
Battery Voltage | `solis_battery_voltage`
Battery Current | `solis_battery_current`
Battery Power | `solis_battery_voltage * solis_battery_current`
Today Energy to Battery | `solis_inverter_energy{period="day",type="charge"}`
Total Energy to Battery | `solis_inverter_energy{period="all",type="charge"}`
Today Energy from Battery | `solis_inverter_energy{period="day",type="discharge"}`
Total Energy from Battery | `solis_inverter_energy{period="all",type="discharge"}`
BMS Battery Voltage | `solis_bms_battery_voltage`
BMS Battery Current | `solis_bms_battery_current`
Battery SOC | `solis_battery_soc`
Battery SOH | `solis_battery_soh`
BMS Battery Charging Current | `solis_bms_charge_limit_current`
BMS Battery Discharge Current | `solis_bms_discharge_limit_current`

<br />
When calculating the power draw from the battery, I find that the default
"Battery Power" metric is sometimes very inaccurate.  In particular, when the
battery has hit its lower limit of 20% and the house is running from grid,
"Battery Current" suggests that around 2.9A (140W) is still being drawn,
whereas "BMS Battery Current" gives a much better estimate of around 0.6A
(30W).  The latter value matches the rate at which I observe the battery
drains (by 1% in ~2 hours, which for my 7.1kWh battery setup is 35W)

"BMS Battery Current" is unsigned (positive for both charging and
discharging), so an alternative battery power drain figure can be calculated
using:

```
solis_bms_battery_current * solis_bms_battery_voltage * sgn(solis_battery_current)
```

"BMS Battery Charging Current" normally sits flat at 74A on my system (i.e.
it is a current *limit*), except when the battery full or nearing full, in
which case it drops down towards zero.

The inverter does provide a battery power figure, at registers 33149-33150;
however, the data logger only reads one half of this 32-bit value.  This
appears to be a bug in the data logger.  Reading it by hand shows that the
value is the same as the product of `solis_battery_voltage` and
`solis_battery_current`.

### Load

Solis Cloud parameter | solis_exporter metric
----------------------|----------------------
Backup AC Voltage | `solis_inverter_backup_voltage`
Backup AC Current | `solis_inverter_backup_current`
Backup Load Power | `solis_inverter_backup_power`
Total Consumption Power | `solis_inverter_load_power`
Today Consumption | `solis_inverter_energy{period="day",type="load"}`
Total Consumption Energy | `solis_inverter_energy{period="all",type="load"}`

<br />
The Consumption Power and Backup Load Power figures seem to be very
unreliable, often dropping to zero despite a near-constant load.  I suspect
these are estimated from inverter output power, grid import/export and
battery charge/discharge.

I find the following formula gives a more consistent estimate of load:

```
clamp_min(0.97 * solis_inverter_dc_power - solis_grid_power_active - 1.06 *
solis_bms_battery_current * solis_bms_battery_voltage * sgn(solis_battery_current), 0)
```

It does show occasional excursions to zero, but much less often, and the
calculated values are plausible.
