= smtptoxmpp =
A small XMPP component to relay emails as XMPP messages.

This project is still in a very early state, so expect emails to be dropped without
warning.

== Configuration ==
smtptoxmpp takes the name of a config file as a single argument; an example follows:

[smtp]
  # hostname to report to smtp clients, defaults to system hostname
  #hostname = "relay"
  port = 25

  [xmpp]
  domain = "localhost"
  name = "smtp" # the name of the component would be smtp.localhost
  secret = "changeme"
  server = "localhost"
  port = 5347
  # inregexp and outregexp are optional, in this example emails are addressed to 
  # the subdomain @xmpp.localhost. The XMPP server only serves @localhost, so 
  # inregexp extracts what lies before the ampersat and outregexp appends the
  # extraction with @locahost. The first pair of () corresponds to $1, the second to
  # $2, and so forth.
  inregexp = "(.*)@xmpp.localhost"
  outregexp = "$1@localhost"

== License ==
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
