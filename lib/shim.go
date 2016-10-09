package launch

import (
	"bytes"
	"fmt"
	"text/template"
)

var shimTmpl = `
var http = require('http');
var spawn = require('child_process').spawn;
var qs = require('querystring');

var waiting = false;
var running = false;

exports.proxy = proxy;

function proxy(event, context) {
	boot(event);
	if (!running) {
		console.log("Proxy: Waiting for application to start.");
		setTimeout(function() {
			proxy(event, context);
		}, 1);
	} else {
		sendRequest(event, context);
	}
}

function sendRequest(event, context) {
	var queries = event.queryStringParameters ? '?' + qs.stringify(event.queryStringParameters) : '';
	var options = {
		port: {{.Port}},
		method: event.method,
		path: event.path + queries,
		headers: event.headers
	};

	var req = http.request(options, function (res) {
		var chunks = [];

		res.on('data', function (data) {
			if (Buffer.isBuffer(data)) {
				chunks.push(data);
			} else {
				chunks.push(new Buffer(data))
			}
		});

		res.on('end', function () {
			context.succeed({
				statusCode: res.statusCode,
				headers: res.headers,
				body: Buffer.concat(chunks).toString()
			});
		});
	});

	if (event.body) {
		req.setHeader('Content-Length', event.body.length);
		req.write(event.body);
	}
	req.end();
}

function boot(event) {
	if (!running && !waiting) {
		waiting = true;
		var server = spawn('./server', [], {env: event.stageVariables});

		server.stdout.on('data', function(data) {
			running = true;
			console.log(String(data));
		});

		server.stderr.on('data', function(data) {
			running = true;
			console.error(String(data));
		});
	}
}`

func Shim(conf *Config) ([]byte, error) {
	buf := new(bytes.Buffer)
	tmpl, err := template.New("shim").Parse(shimTmpl)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse shim template: %v", err)
	}

	if err := tmpl.Execute(buf, conf); err != nil {
		return nil, fmt.Errorf("Unable to generate shim: %v", err)
	}

	return buf.Bytes(), nil
}
