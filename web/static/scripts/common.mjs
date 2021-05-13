// Copyright 2020-2021 The OS-NVR Authors.
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation; version 2.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

function $(query) {
	return document.querySelector(query);
}
function $$(query) {
	return document.querySelectorAll(query);
}

async function sendAlert(msg, response) {
	alert(`${msg}: ${response.status}, ${await response.text()}`);
}

async function fetchGet(url, msg) {
	const response = await fetch(url, { method: "get" });
	if (response.status !== 200) {
		sendAlert(msg, await response);
		return;
	}
	return await response.json();
}

async function fetchPost(url, data, token, msg) {
	const response = await fetch(url, {
		body: JSON.stringify(data),
		headers: {
			"Content-Type": "application/json",
			"X-CSRF-TOKEN": token,
		},
		method: "post",
	});
	if (response.status !== 200) {
		sendAlert(msg, response);
		return false;
	}
	return true;
}

async function fetchPut(url, data, token, msg) {
	const response = await fetch(url, {
		body: JSON.stringify(data),
		headers: {
			"Content-Type": "application/json",
			"X-CSRF-TOKEN": token,
		},
		method: "put",
	});
	if (response.status !== 200) {
		sendAlert(msg, response);
		return false;
	}
	return true;
}

async function fetchDelete(url, token, msg) {
	const response = await fetch(url, {
		headers: {
			"X-CSRF-TOKEN": token,
		},
		method: "delete",
	});
	if (response.status !== 200) {
		sendAlert(msg, response);
		return false;
	}
	return true;
}

export { $, $$, fetchGet, fetchPost, fetchPut, fetchDelete };