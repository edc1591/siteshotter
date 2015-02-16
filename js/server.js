var host, port, server, service,
  system = require('system'),
  webPage = require('webpage');

if (system.args.length !== 3) {
  console.log('phantomjs: Usage: server.js <host> <portnumber>');
  phantom.exit(1);
}

host = system.args[1];
port = system.args[2];
server = require('webserver').create();

service = server.listen(port, { keepAlive: true }, function (request, response) {
  response.headers = {
    'Cache': 'no-cache',
    'Content-Type': 'text/plain',
    'Connection': 'Keep-Alive',
    'Keep-Alive': 'timeout=5, max=100',
    'Content-Length': 0
  };

  if (request.url === '/ping') {
    response.statusCode = 200;
    response.closeGracefully();
    return;
  }
  
  if (request.url.slice(0, '/image.png'.length) !== '/image.png') {
    response.statusCode = 404;
    response.closeGracefully();
    return;
  }

  var params = {};
  if (request.url.slice('/image.png'.length)[0] === "?") {
    query = request.url.slice('/image.png?'.length);
    query.split('&').forEach(function (pair) { 
      var p = pair.split('='); 
      params[p[0]] = p[1];
    });
  }

  render(params.url, params.selector, function (base64Image) {
    var body = atob(base64Image);
    response.statusCode = 200;
    response.setHeader('Content-Length', body.length);
    response.setHeader('Content-Type', 'image/png');
    response.setEncoding('binary');
    response.write(body);
    response.close();
  }, function (error) {
    console.log('phantomjs: ['+port+'] error: ' + error);
    response.statusCode = 500;
    response.setHeader('Content-Length', error.length);
    response.write(error);
    response.close();
  });
});

if (service) {
  console.log('phantomjs: Web server running on port ' + port);
} else {
  console.log('phantomjs: Error: Could not create web server listening on port ' + port);
  phantom.exit();
}

function render(url, selector, success, failure) {
  var page = webPage.create();
  page.zoomFactor = 1;
  page.settings.javascriptEnabled = true;

  if (selector) {
    var clipRect = page.evaluate(function (selector) {
      console.log("selector: " + s);
      return document.querySelector(s).getBoundingClientRect();
    }, { selector: selector });
    console.log('hi');
    console.log(JSON.stringify(clipRect));
    page.clipRect = {
      top:    clipRect.top,
      left:   clipRect.left,
      width:  clipRect.width,
      height: clipRect.height
    };
  }
  page.open(decodeURI(url), function (status) {
    if (status !== 'success') {
      failure(status + ' unable to load the permalink');
      page.close();
      return;
    }
    var start = Date.now();
    var base64 = page.renderBase64('PNG');
    console.log('phantomjs: ['+port+'] render ' + ((Date.now() - start) / 1000) + 's');
    page.close();
    success(base64);
  });
}