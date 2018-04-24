'use strict';

require('libsys/shim')
require('libjs/shim')
var script = require('/function/script.js')

var qfd = parseInt(process.argv[2], 10)

var fd = parseInt(process.argv[3], 10)
libsys.syscall(33, fd, 1)
libjs.close(fd)

var fd = parseInt(process.argv[4], 10)
libsys.syscall(33, fd, 2)
libjs.close(fd)

var buf = Buffer.alloc(1024)

function recv() {
	var ret = ""

	for (;;) {
		var l = libjs.recv(qfd, buf)

		ret += buf.toString('utf8', 0, l)
		if (l < 1024)
			return ret
	}
}

function send(res) {
	if (res.length % 1024 == 0)
		res += "\0"

	while (res.length > 0) {
		var s = res.substring(0, 1024)
		libjs.send(qfd, s)
		res = res.substring(1024)
	}
}

for (;;) {
	var req = recv()
	var args = JSON.parse(req)
	var res
	try {
		var ret = script.Main(args)
		res = "0:" + JSON.stringify(ret)
	} catch (err) {
		res = "1:Exception"
	}

	send(res)
}

process.exit(1)