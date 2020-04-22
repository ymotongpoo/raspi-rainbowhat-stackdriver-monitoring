# raspi-rainbowhat-stackdriver-monitoring
This sample fetches temperature and pressure from BMP280 on Rainbow HAT,
and sends those data to Stackdriver Monitoring via OpenCensus.

## Reference
### BMP280 on Rainbow HAT

* Rainbow HAT
  * [Pimoroni](https://shop.pimoroni.com/products/rainbow-hat-for-android-things)
  * [SwitchScience](https://www.switch-science.com/catalog/3225/)
* [periph](https://periph.io/)
  * [periph - rainbowhat](https://github.com/google/periph/tree/master/experimental/devices/rainbowhat)

### OpenCensus Stats + Stackdriver Monitoring
* [OpenCensus Metrics Quickstart for Go](https://opencensus.io/quickstart/go/metrics/)
* [OpenCensus Data Aggregation API Overview](https://github.com/census-instrumentation/opencensus-specs/blob/master/stats/DataAggregation.md)
* [Sample OpenCensus Views and Metrics for gRPC](https://godoc.org/go.opencensus.io/plugin/ocgrpc#pkg-variables)
* GoDoc
  * https://godoc.org/go.opencensus.io/stats
  * https://godoc.org/go.opencensus.io/stats/view
  * https://godoc.org/contrib.go.opencensus.io/exporter/stackdriver
  * https://godoc.org/contrib.go.opencensus.io/exporter/stackdriver/monitoredresource
* [OpenCensus and Stackdriver Monitoring Terminology and Concepts](https://cloud.google.com/monitoring/custom-metrics/open-census#opencensus-vocabulary)