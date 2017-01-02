meetup_client_go

API consumer library for interacting with the Meetup API.

This repo is still a work in progress. Any contributions are welcome.

Features:

1. Get Venues by group url
2. Get Events by Venue
3. Get Events by group
4. Get Events (v3 events from Meetup)

There are a bunch of GetAllxxx APIs, which recursively query all the events/venues.
Output will be a list of page results (2d array).

Not all paramemters from the meetup API are supported.

Tests are not implemented yet.
