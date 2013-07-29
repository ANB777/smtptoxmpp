# smtptoxmpp
A small XMPP component to relay emails as XMPP messages.

smtptoxmpp must be used with Inetd or Systemd.

Be warned if an email is sent to an address for which there is no XMPP account, 
it is dropped without error.

## Configuration
smtptoxmpp takes the name of a config file as a single argument; an example follows:

    [xmpp]
    domain = "example.com"
    name = "smtp" # the name of the component would be smtp.example.com
    secret = "changeme"
    server = "example.com"
    port = 5347
    # inregexp and outregexp are optional, in this example emails are addressed to 
    # the subdomain @xmpp.example.com. The XMPP server only serves @example.com, so 
    # inregexp extracts what lies before the ampersat and outregexp appends the
    # extraction with @locahost. The first pair of () corresponds to $1, the second
    # to $2, and so forth.
    inregexp = "(.*)@xmpp.example.com"
    outregexp = "$1@example.com"

The option flag -idle will set the idle period after which the process will die. 
Use a time like 30s or 2m, the default is one minute (1m).

## Serving a sub-domain on the same machine as Postfix
Add this to /etc/postfix/main.cf
    transport_maps = hash:/etc/postfix/transport
Add a line like this to /etc/postfix/transport
    xmpp.example.com       smtp:[localhost]:5225
Then set inetd or systemd to activate smtptoxmpp on port 5225

## FAQ
#### Email isn't broken, yur dumb.
* Fact: [Email is almost entirely garbage](https://en.wikipedia.org/wiki/Email_spam#Statistics_and_estimates).
* Fact: The hundreds if not thousands of emails you have sitting server-side aren't 
actually worth saving. If you can't remember what they were about after five minutes 
either they weren't important or you're doing something wrong.

#### Why wont it run as a standalone daemon?
Messages will either queue at the SMTP client or the XMPP destination. There is no
point in running an idle process when you can easily start it on demand. It's also 
easier to define socket activation than to define init system specifications for 
each distro.

## Donations
[1M2dJsxA2J8ayG7xqGP5Rg1KeWS3CGxxbZ](bitcoin:1M2dJsxA2J8ayG7xqGP5Rg1KeWS3CGxxbZ)

## License
Copyright (C) 2013 Emery Hemingway xmpp:emery@fuzzlabs.org

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
