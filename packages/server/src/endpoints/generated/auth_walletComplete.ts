export const auth_walletComplete = {
  "fieldName": "walletComplete",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "walletComplete",
        "loc": {
          "start": 5327,
          "end": 5341
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 5342,
              "end": 5347
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 5350,
                "end": 5355
              }
            },
            "loc": {
              "start": 5349,
              "end": 5355
            }
          },
          "loc": {
            "start": 5342,
            "end": 5355
          }
        }
      ],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "firstLogIn",
              "loc": {
                "start": 5363,
                "end": 5373
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5363,
              "end": 5373
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "session",
              "loc": {
                "start": 5378,
                "end": 5385
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Session_full",
                    "loc": {
                      "start": 5399,
                      "end": 5411
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5396,
                    "end": 5411
                  }
                }
              ],
              "loc": {
                "start": 5386,
                "end": 5417
              }
            },
            "loc": {
              "start": 5378,
              "end": 5417
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wallet",
              "loc": {
                "start": 5422,
                "end": 5428
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "FragmentSpread",
                  "name": {
                    "kind": "Name",
                    "value": "Wallet_common",
                    "loc": {
                      "start": 5442,
                      "end": 5455
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 5439,
                    "end": 5455
                  }
                }
              ],
              "loc": {
                "start": 5429,
                "end": 5461
              }
            },
            "loc": {
              "start": 5422,
              "end": 5461
            }
          }
        ],
        "loc": {
          "start": 5357,
          "end": 5465
        }
      },
      "loc": {
        "start": 5327,
        "end": 5465
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 40,
          "end": 42
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 40,
        "end": 42
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "created_at",
        "loc": {
          "start": 43,
          "end": 53
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 43,
        "end": 53
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "updated_at",
        "loc": {
          "start": 54,
          "end": 64
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 54,
        "end": 64
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "startTime",
        "loc": {
          "start": 65,
          "end": 74
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 65,
        "end": 74
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "endTime",
        "loc": {
          "start": 75,
          "end": 82
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 75,
        "end": 82
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timezone",
        "loc": {
          "start": 83,
          "end": 91
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 83,
        "end": 91
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "exceptions",
        "loc": {
          "start": 92,
          "end": 102
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 109,
                "end": 111
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 109,
              "end": 111
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "originalStartTime",
              "loc": {
                "start": 116,
                "end": 133
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 116,
              "end": 133
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newStartTime",
              "loc": {
                "start": 138,
                "end": 150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 138,
              "end": 150
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "newEndTime",
              "loc": {
                "start": 155,
                "end": 165
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 155,
              "end": 165
            }
          }
        ],
        "loc": {
          "start": 103,
          "end": 167
        }
      },
      "loc": {
        "start": 92,
        "end": 167
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "recurrences",
        "loc": {
          "start": 168,
          "end": 179
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 186,
                "end": 188
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 186,
              "end": 188
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrenceType",
              "loc": {
                "start": 193,
                "end": 207
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 193,
              "end": 207
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "interval",
              "loc": {
                "start": 212,
                "end": 220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 212,
              "end": 220
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfWeek",
              "loc": {
                "start": 225,
                "end": 234
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 225,
              "end": 234
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "dayOfMonth",
              "loc": {
                "start": 239,
                "end": 249
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 239,
              "end": 249
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "month",
              "loc": {
                "start": 254,
                "end": 259
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 254,
              "end": 259
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endDate",
              "loc": {
                "start": 264,
                "end": 271
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 264,
              "end": 271
            }
          }
        ],
        "loc": {
          "start": 180,
          "end": 273
        }
      },
      "loc": {
        "start": 168,
        "end": 273
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "isLoggedIn",
        "loc": {
          "start": 311,
          "end": 321
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 311,
        "end": 321
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "timeZone",
        "loc": {
          "start": 322,
          "end": 330
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 322,
        "end": 330
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "users",
        "loc": {
          "start": 331,
          "end": 336
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "activeFocusMode",
              "loc": {
                "start": 343,
                "end": 358
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "mode",
                    "loc": {
                      "start": 369,
                      "end": 373
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 388,
                            "end": 395
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 414,
                                  "end": 416
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 414,
                                "end": 416
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 433,
                                  "end": 443
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 433,
                                "end": 443
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 460,
                                  "end": 463
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 486,
                                        "end": 488
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 486,
                                      "end": 488
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 509,
                                        "end": 519
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 509,
                                      "end": 519
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 540,
                                        "end": 543
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 540,
                                      "end": 543
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 564,
                                        "end": 573
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 564,
                                      "end": 573
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 594,
                                        "end": 606
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 633,
                                              "end": 635
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 633,
                                            "end": 635
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 660,
                                              "end": 668
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 660,
                                            "end": 668
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 693,
                                              "end": 704
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 693,
                                            "end": 704
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 607,
                                        "end": 726
                                      }
                                    },
                                    "loc": {
                                      "start": 594,
                                      "end": 726
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 747,
                                        "end": 750
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 777,
                                              "end": 782
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 777,
                                            "end": 782
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 807,
                                              "end": 819
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 807,
                                            "end": 819
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 751,
                                        "end": 841
                                      }
                                    },
                                    "loc": {
                                      "start": 747,
                                      "end": 841
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 464,
                                  "end": 859
                                }
                              },
                              "loc": {
                                "start": 460,
                                "end": 859
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 876,
                                  "end": 885
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 908,
                                        "end": 914
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 941,
                                              "end": 943
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 941,
                                            "end": 943
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 968,
                                              "end": 973
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 968,
                                            "end": 973
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 998,
                                              "end": 1003
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 998,
                                            "end": 1003
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 915,
                                        "end": 1025
                                      }
                                    },
                                    "loc": {
                                      "start": 908,
                                      "end": 1025
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 1046,
                                        "end": 1054
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 1084,
                                              "end": 1099
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 1081,
                                            "end": 1099
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1055,
                                        "end": 1121
                                      }
                                    },
                                    "loc": {
                                      "start": 1046,
                                      "end": 1121
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1142,
                                        "end": 1144
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1142,
                                      "end": 1144
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 1165,
                                        "end": 1169
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1165,
                                      "end": 1169
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 1190,
                                        "end": 1201
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1190,
                                      "end": 1201
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 886,
                                  "end": 1219
                                }
                              },
                              "loc": {
                                "start": 876,
                                "end": 1219
                              }
                            }
                          ],
                          "loc": {
                            "start": 396,
                            "end": 1233
                          }
                        },
                        "loc": {
                          "start": 388,
                          "end": 1233
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 1246,
                            "end": 1252
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1271,
                                  "end": 1273
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1271,
                                "end": 1273
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 1290,
                                  "end": 1295
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1290,
                                "end": 1295
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 1312,
                                  "end": 1317
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1312,
                                "end": 1317
                              }
                            }
                          ],
                          "loc": {
                            "start": 1253,
                            "end": 1331
                          }
                        },
                        "loc": {
                          "start": 1246,
                          "end": 1331
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 1344,
                            "end": 1356
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 1375,
                                  "end": 1377
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1375,
                                "end": 1377
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 1394,
                                  "end": 1404
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1394,
                                "end": 1404
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 1421,
                                  "end": 1431
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 1421,
                                "end": 1431
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 1448,
                                  "end": 1457
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1480,
                                        "end": 1482
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1480,
                                      "end": 1482
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1503,
                                        "end": 1513
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1503,
                                      "end": 1513
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1534,
                                        "end": 1544
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1534,
                                      "end": 1544
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 1565,
                                        "end": 1569
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1565,
                                      "end": 1569
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 1590,
                                        "end": 1601
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1590,
                                      "end": 1601
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 1622,
                                        "end": 1629
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1622,
                                      "end": 1629
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 1650,
                                        "end": 1655
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1650,
                                      "end": 1655
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 1676,
                                        "end": 1686
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1676,
                                      "end": 1686
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 1707,
                                        "end": 1720
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1747,
                                              "end": 1749
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1747,
                                            "end": 1749
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1774,
                                              "end": 1784
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1774,
                                            "end": 1784
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1809,
                                              "end": 1819
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1809,
                                            "end": 1819
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1844,
                                              "end": 1848
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1844,
                                            "end": 1848
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1873,
                                              "end": 1884
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1873,
                                            "end": 1884
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 1909,
                                              "end": 1916
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1909,
                                            "end": 1916
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 1941,
                                              "end": 1946
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1941,
                                            "end": 1946
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 1971,
                                              "end": 1981
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1971,
                                            "end": 1981
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1721,
                                        "end": 2003
                                      }
                                    },
                                    "loc": {
                                      "start": 1707,
                                      "end": 2003
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1458,
                                  "end": 2021
                                }
                              },
                              "loc": {
                                "start": 1448,
                                "end": 2021
                              }
                            }
                          ],
                          "loc": {
                            "start": 1357,
                            "end": 2035
                          }
                        },
                        "loc": {
                          "start": 1344,
                          "end": 2035
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 2048,
                            "end": 2060
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2079,
                                  "end": 2081
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2079,
                                "end": 2081
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 2098,
                                  "end": 2108
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2098,
                                "end": 2108
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 2125,
                                  "end": 2137
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2160,
                                        "end": 2162
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2160,
                                      "end": 2162
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 2183,
                                        "end": 2191
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2183,
                                      "end": 2191
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2212,
                                        "end": 2223
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2212,
                                      "end": 2223
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2244,
                                        "end": 2248
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2244,
                                      "end": 2248
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2138,
                                  "end": 2266
                                }
                              },
                              "loc": {
                                "start": 2125,
                                "end": 2266
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 2283,
                                  "end": 2292
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2315,
                                        "end": 2317
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2315,
                                      "end": 2317
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 2338,
                                        "end": 2343
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2338,
                                      "end": 2343
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 2364,
                                        "end": 2368
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2364,
                                      "end": 2368
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 2389,
                                        "end": 2396
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2389,
                                      "end": 2396
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2417,
                                        "end": 2429
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2456,
                                              "end": 2458
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2456,
                                            "end": 2458
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2483,
                                              "end": 2491
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2483,
                                            "end": 2491
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2516,
                                              "end": 2527
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2516,
                                            "end": 2527
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2552,
                                              "end": 2556
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2552,
                                            "end": 2556
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2430,
                                        "end": 2578
                                      }
                                    },
                                    "loc": {
                                      "start": 2417,
                                      "end": 2578
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2293,
                                  "end": 2596
                                }
                              },
                              "loc": {
                                "start": 2283,
                                "end": 2596
                              }
                            }
                          ],
                          "loc": {
                            "start": 2061,
                            "end": 2610
                          }
                        },
                        "loc": {
                          "start": 2048,
                          "end": 2610
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 2623,
                            "end": 2631
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 2653,
                                  "end": 2668
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 2650,
                                "end": 2668
                              }
                            }
                          ],
                          "loc": {
                            "start": 2632,
                            "end": 2682
                          }
                        },
                        "loc": {
                          "start": 2623,
                          "end": 2682
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2695,
                            "end": 2697
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2695,
                          "end": 2697
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 2710,
                            "end": 2714
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2710,
                          "end": 2714
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 2727,
                            "end": 2738
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2727,
                          "end": 2738
                        }
                      }
                    ],
                    "loc": {
                      "start": 374,
                      "end": 2748
                    }
                  },
                  "loc": {
                    "start": 369,
                    "end": 2748
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stopCondition",
                    "loc": {
                      "start": 2757,
                      "end": 2770
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2757,
                    "end": 2770
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stopTime",
                    "loc": {
                      "start": 2779,
                      "end": 2787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2779,
                    "end": 2787
                  }
                }
              ],
              "loc": {
                "start": 359,
                "end": 2793
              }
            },
            "loc": {
              "start": 343,
              "end": 2793
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apisCount",
              "loc": {
                "start": 2798,
                "end": 2807
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 2798,
              "end": 2807
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarkLists",
              "loc": {
                "start": 2812,
                "end": 2825
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 2836,
                      "end": 2838
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2836,
                    "end": 2838
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 2847,
                      "end": 2857
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2847,
                    "end": 2857
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 2866,
                      "end": 2876
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2866,
                    "end": 2876
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 2885,
                      "end": 2890
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2885,
                    "end": 2890
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarksCount",
                    "loc": {
                      "start": 2899,
                      "end": 2913
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2899,
                    "end": 2913
                  }
                }
              ],
              "loc": {
                "start": 2826,
                "end": 2919
              }
            },
            "loc": {
              "start": 2812,
              "end": 2919
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 2924,
                "end": 2934
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "filters",
                    "loc": {
                      "start": 2945,
                      "end": 2952
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2967,
                            "end": 2969
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2967,
                          "end": 2969
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filterType",
                          "loc": {
                            "start": 2982,
                            "end": 2992
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2982,
                          "end": 2992
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "tag",
                          "loc": {
                            "start": 3005,
                            "end": 3008
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3027,
                                  "end": 3029
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3027,
                                "end": 3029
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3046,
                                  "end": 3056
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3046,
                                "end": 3056
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 3073,
                                  "end": 3076
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3073,
                                "end": 3076
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bookmarks",
                                "loc": {
                                  "start": 3093,
                                  "end": 3102
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3093,
                                "end": 3102
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 3119,
                                  "end": 3131
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3154,
                                        "end": 3156
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3154,
                                      "end": 3156
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 3177,
                                        "end": 3185
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3177,
                                      "end": 3185
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3206,
                                        "end": 3217
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3206,
                                      "end": 3217
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3132,
                                  "end": 3235
                                }
                              },
                              "loc": {
                                "start": 3119,
                                "end": 3235
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 3252,
                                  "end": 3255
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isOwn",
                                      "loc": {
                                        "start": 3278,
                                        "end": 3283
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3278,
                                      "end": 3283
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 3304,
                                        "end": 3316
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3304,
                                      "end": 3316
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3256,
                                  "end": 3334
                                }
                              },
                              "loc": {
                                "start": 3252,
                                "end": 3334
                              }
                            }
                          ],
                          "loc": {
                            "start": 3009,
                            "end": 3348
                          }
                        },
                        "loc": {
                          "start": 3005,
                          "end": 3348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "focusMode",
                          "loc": {
                            "start": 3361,
                            "end": 3370
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 3389,
                                  "end": 3395
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3418,
                                        "end": 3420
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3418,
                                      "end": 3420
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 3441,
                                        "end": 3446
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3441,
                                      "end": 3446
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 3467,
                                        "end": 3472
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3467,
                                      "end": 3472
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3396,
                                  "end": 3490
                                }
                              },
                              "loc": {
                                "start": 3389,
                                "end": 3490
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 3507,
                                  "end": 3515
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 3541,
                                        "end": 3556
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 3538,
                                      "end": 3556
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3516,
                                  "end": 3574
                                }
                              },
                              "loc": {
                                "start": 3507,
                                "end": 3574
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3591,
                                  "end": 3593
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3591,
                                "end": 3593
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3610,
                                  "end": 3614
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3610,
                                "end": 3614
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3631,
                                  "end": 3642
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3631,
                                "end": 3642
                              }
                            }
                          ],
                          "loc": {
                            "start": 3371,
                            "end": 3656
                          }
                        },
                        "loc": {
                          "start": 3361,
                          "end": 3656
                        }
                      }
                    ],
                    "loc": {
                      "start": 2953,
                      "end": 3666
                    }
                  },
                  "loc": {
                    "start": 2945,
                    "end": 3666
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 3675,
                      "end": 3681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3696,
                            "end": 3698
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3696,
                          "end": 3698
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 3711,
                            "end": 3716
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3711,
                          "end": 3716
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 3729,
                            "end": 3734
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3729,
                          "end": 3734
                        }
                      }
                    ],
                    "loc": {
                      "start": 3682,
                      "end": 3744
                    }
                  },
                  "loc": {
                    "start": 3675,
                    "end": 3744
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 3753,
                      "end": 3765
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 3780,
                            "end": 3782
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3780,
                          "end": 3782
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 3795,
                            "end": 3805
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3795,
                          "end": 3805
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 3818,
                            "end": 3828
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 3818,
                          "end": 3828
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 3841,
                            "end": 3850
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3869,
                                  "end": 3871
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3869,
                                "end": 3871
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3888,
                                  "end": 3898
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3888,
                                "end": 3898
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3915,
                                  "end": 3925
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3915,
                                "end": 3925
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 3942,
                                  "end": 3946
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3942,
                                "end": 3946
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 3963,
                                  "end": 3974
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3963,
                                "end": 3974
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 3991,
                                  "end": 3998
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3991,
                                "end": 3998
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 4015,
                                  "end": 4020
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4015,
                                "end": 4020
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 4037,
                                  "end": 4047
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4037,
                                "end": 4047
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 4064,
                                  "end": 4077
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4100,
                                        "end": 4102
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4100,
                                      "end": 4102
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4123,
                                        "end": 4133
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4123,
                                      "end": 4133
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 4154,
                                        "end": 4164
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4154,
                                      "end": 4164
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4185,
                                        "end": 4189
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4185,
                                      "end": 4189
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4210,
                                        "end": 4221
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4210,
                                      "end": 4221
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 4242,
                                        "end": 4249
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4242,
                                      "end": 4249
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4270,
                                        "end": 4275
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4270,
                                      "end": 4275
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 4296,
                                        "end": 4306
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4296,
                                      "end": 4306
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4078,
                                  "end": 4324
                                }
                              },
                              "loc": {
                                "start": 4064,
                                "end": 4324
                              }
                            }
                          ],
                          "loc": {
                            "start": 3851,
                            "end": 4338
                          }
                        },
                        "loc": {
                          "start": 3841,
                          "end": 4338
                        }
                      }
                    ],
                    "loc": {
                      "start": 3766,
                      "end": 4348
                    }
                  },
                  "loc": {
                    "start": 3753,
                    "end": 4348
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 4357,
                      "end": 4369
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4384,
                            "end": 4386
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4384,
                          "end": 4386
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4399,
                            "end": 4409
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4399,
                          "end": 4409
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 4422,
                            "end": 4434
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4453,
                                  "end": 4455
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4453,
                                "end": 4455
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 4472,
                                  "end": 4480
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4472,
                                "end": 4480
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 4497,
                                  "end": 4508
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4497,
                                "end": 4508
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4525,
                                  "end": 4529
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4525,
                                "end": 4529
                              }
                            }
                          ],
                          "loc": {
                            "start": 4435,
                            "end": 4543
                          }
                        },
                        "loc": {
                          "start": 4422,
                          "end": 4543
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 4556,
                            "end": 4565
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4584,
                                  "end": 4586
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4584,
                                "end": 4586
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 4603,
                                  "end": 4608
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4603,
                                "end": 4608
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 4625,
                                  "end": 4629
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4625,
                                "end": 4629
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 4646,
                                  "end": 4653
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4646,
                                "end": 4653
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4670,
                                  "end": 4682
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4705,
                                        "end": 4707
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4705,
                                      "end": 4707
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4728,
                                        "end": 4736
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4728,
                                      "end": 4736
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4757,
                                        "end": 4768
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4757,
                                      "end": 4768
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4789,
                                        "end": 4793
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4789,
                                      "end": 4793
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4683,
                                  "end": 4811
                                }
                              },
                              "loc": {
                                "start": 4670,
                                "end": 4811
                              }
                            }
                          ],
                          "loc": {
                            "start": 4566,
                            "end": 4825
                          }
                        },
                        "loc": {
                          "start": 4556,
                          "end": 4825
                        }
                      }
                    ],
                    "loc": {
                      "start": 4370,
                      "end": 4835
                    }
                  },
                  "loc": {
                    "start": 4357,
                    "end": 4835
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 4844,
                      "end": 4852
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "FragmentSpread",
                        "name": {
                          "kind": "Name",
                          "value": "Schedule_common",
                          "loc": {
                            "start": 4870,
                            "end": 4885
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 4867,
                          "end": 4885
                        }
                      }
                    ],
                    "loc": {
                      "start": 4853,
                      "end": 4895
                    }
                  },
                  "loc": {
                    "start": 4844,
                    "end": 4895
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4904,
                      "end": 4906
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4904,
                    "end": 4906
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 4915,
                      "end": 4919
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4915,
                    "end": 4919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 4928,
                      "end": 4939
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4928,
                    "end": 4939
                  }
                }
              ],
              "loc": {
                "start": 2935,
                "end": 4945
              }
            },
            "loc": {
              "start": 2924,
              "end": 4945
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 4950,
                "end": 4956
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4950,
              "end": 4956
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasPremium",
              "loc": {
                "start": 4961,
                "end": 4971
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4961,
              "end": 4971
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 4976,
                "end": 4978
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4976,
              "end": 4978
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "languages",
              "loc": {
                "start": 4983,
                "end": 4992
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4983,
              "end": 4992
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membershipsCount",
              "loc": {
                "start": 4997,
                "end": 5013
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4997,
              "end": 5013
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 5018,
                "end": 5022
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5018,
              "end": 5022
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 5027,
                "end": 5037
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5027,
              "end": 5037
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 5042,
                "end": 5055
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5042,
              "end": 5055
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsAskedCount",
              "loc": {
                "start": 5060,
                "end": 5079
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5060,
              "end": 5079
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 5084,
                "end": 5097
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5084,
              "end": 5097
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "smartContractsCount",
              "loc": {
                "start": 5102,
                "end": 5121
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5102,
              "end": 5121
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 5126,
                "end": 5140
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5126,
              "end": 5140
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "theme",
              "loc": {
                "start": 5145,
                "end": 5150
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5145,
              "end": 5150
            }
          }
        ],
        "loc": {
          "start": 337,
          "end": 5152
        }
      },
      "loc": {
        "start": 331,
        "end": 5152
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 5190,
          "end": 5192
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5190,
        "end": 5192
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "handles",
        "loc": {
          "start": 5193,
          "end": 5200
        }
      },
      "arguments": [],
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5207,
                "end": 5209
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5207,
              "end": 5209
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 5214,
                "end": 5220
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5214,
              "end": 5220
            }
          }
        ],
        "loc": {
          "start": 5201,
          "end": 5222
        }
      },
      "loc": {
        "start": 5193,
        "end": 5222
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 5223,
          "end": 5227
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5223,
        "end": 5227
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "publicAddress",
        "loc": {
          "start": 5228,
          "end": 5241
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5228,
        "end": 5241
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stakingAddress",
        "loc": {
          "start": 5242,
          "end": 5256
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5242,
        "end": 5256
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "verified",
        "loc": {
          "start": 5257,
          "end": 5265
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 5257,
        "end": 5265
      }
    }
  ],
  "returnType": null,
  "parentType": null,
  "schema": null,
  "fragments": {
    "Schedule_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Schedule_common",
        "loc": {
          "start": 10,
          "end": 25
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Schedule",
          "loc": {
            "start": 29,
            "end": 37
          }
        },
        "loc": {
          "start": 29,
          "end": 37
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 40,
                "end": 42
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 40,
              "end": 42
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "created_at",
              "loc": {
                "start": 43,
                "end": 53
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 43,
              "end": 53
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "updated_at",
              "loc": {
                "start": 54,
                "end": 64
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 54,
              "end": 64
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "startTime",
              "loc": {
                "start": 65,
                "end": 74
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 65,
              "end": 74
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "endTime",
              "loc": {
                "start": 75,
                "end": 82
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 75,
              "end": 82
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timezone",
              "loc": {
                "start": 83,
                "end": 91
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 83,
              "end": 91
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "exceptions",
              "loc": {
                "start": 92,
                "end": 102
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 109,
                      "end": 111
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 109,
                    "end": 111
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "originalStartTime",
                    "loc": {
                      "start": 116,
                      "end": 133
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 116,
                    "end": 133
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newStartTime",
                    "loc": {
                      "start": 138,
                      "end": 150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 138,
                    "end": 150
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "newEndTime",
                    "loc": {
                      "start": 155,
                      "end": 165
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 155,
                    "end": 165
                  }
                }
              ],
              "loc": {
                "start": 103,
                "end": 167
              }
            },
            "loc": {
              "start": 92,
              "end": 167
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "recurrences",
              "loc": {
                "start": 168,
                "end": 179
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 186,
                      "end": 188
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 186,
                    "end": 188
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "recurrenceType",
                    "loc": {
                      "start": 193,
                      "end": 207
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 193,
                    "end": 207
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "interval",
                    "loc": {
                      "start": 212,
                      "end": 220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 212,
                    "end": 220
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfWeek",
                    "loc": {
                      "start": 225,
                      "end": 234
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 225,
                    "end": 234
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "dayOfMonth",
                    "loc": {
                      "start": 239,
                      "end": 249
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 239,
                    "end": 249
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "month",
                    "loc": {
                      "start": 254,
                      "end": 259
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 254,
                    "end": 259
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "endDate",
                    "loc": {
                      "start": 264,
                      "end": 271
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 264,
                    "end": 271
                  }
                }
              ],
              "loc": {
                "start": 180,
                "end": 273
              }
            },
            "loc": {
              "start": 168,
              "end": 273
            }
          }
        ],
        "loc": {
          "start": 38,
          "end": 275
        }
      },
      "loc": {
        "start": 1,
        "end": 275
      }
    },
    "Session_full": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Session_full",
        "loc": {
          "start": 285,
          "end": 297
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Session",
          "loc": {
            "start": 301,
            "end": 308
          }
        },
        "loc": {
          "start": 301,
          "end": 308
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "isLoggedIn",
              "loc": {
                "start": 311,
                "end": 321
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 311,
              "end": 321
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 322,
                "end": 330
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 322,
              "end": 330
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 331,
                "end": 336
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "activeFocusMode",
                    "loc": {
                      "start": 343,
                      "end": 358
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "mode",
                          "loc": {
                            "start": 369,
                            "end": 373
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filters",
                                "loc": {
                                  "start": 388,
                                  "end": 395
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 414,
                                        "end": 416
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 414,
                                      "end": 416
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 433,
                                        "end": 443
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 433,
                                      "end": 443
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 460,
                                        "end": 463
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 486,
                                              "end": 488
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 486,
                                            "end": 488
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 509,
                                              "end": 519
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 509,
                                            "end": 519
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 540,
                                              "end": 543
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 540,
                                            "end": 543
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 564,
                                              "end": 573
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 564,
                                            "end": 573
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 594,
                                              "end": 606
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 633,
                                                    "end": 635
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 633,
                                                  "end": 635
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 660,
                                                    "end": 668
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 660,
                                                  "end": 668
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 693,
                                                    "end": 704
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 693,
                                                  "end": 704
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 607,
                                              "end": 726
                                            }
                                          },
                                          "loc": {
                                            "start": 594,
                                            "end": 726
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 747,
                                              "end": 750
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isOwn",
                                                  "loc": {
                                                    "start": 777,
                                                    "end": 782
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 777,
                                                  "end": 782
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 807,
                                                    "end": 819
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 807,
                                                  "end": 819
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 751,
                                              "end": 841
                                            }
                                          },
                                          "loc": {
                                            "start": 747,
                                            "end": 841
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 464,
                                        "end": 859
                                      }
                                    },
                                    "loc": {
                                      "start": 460,
                                      "end": 859
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 876,
                                        "end": 885
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "labels",
                                            "loc": {
                                              "start": 908,
                                              "end": 914
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 941,
                                                    "end": 943
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 941,
                                                  "end": 943
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 968,
                                                    "end": 973
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 968,
                                                  "end": 973
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 998,
                                                    "end": 1003
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 998,
                                                  "end": 1003
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 915,
                                              "end": 1025
                                            }
                                          },
                                          "loc": {
                                            "start": 908,
                                            "end": 1025
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 1046,
                                              "end": 1054
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "FragmentSpread",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "Schedule_common",
                                                  "loc": {
                                                    "start": 1084,
                                                    "end": 1099
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 1081,
                                                  "end": 1099
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1055,
                                              "end": 1121
                                            }
                                          },
                                          "loc": {
                                            "start": 1046,
                                            "end": 1121
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1142,
                                              "end": 1144
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1142,
                                            "end": 1144
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1165,
                                              "end": 1169
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1165,
                                            "end": 1169
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1190,
                                              "end": 1201
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1190,
                                            "end": 1201
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 886,
                                        "end": 1219
                                      }
                                    },
                                    "loc": {
                                      "start": 876,
                                      "end": 1219
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 396,
                                  "end": 1233
                                }
                              },
                              "loc": {
                                "start": 388,
                                "end": 1233
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 1246,
                                  "end": 1252
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1271,
                                        "end": 1273
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1271,
                                      "end": 1273
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 1290,
                                        "end": 1295
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1290,
                                      "end": 1295
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 1312,
                                        "end": 1317
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1312,
                                      "end": 1317
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1253,
                                  "end": 1331
                                }
                              },
                              "loc": {
                                "start": 1246,
                                "end": 1331
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 1344,
                                  "end": 1356
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 1375,
                                        "end": 1377
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1375,
                                      "end": 1377
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 1394,
                                        "end": 1404
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1394,
                                      "end": 1404
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 1421,
                                        "end": 1431
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 1421,
                                      "end": 1431
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 1448,
                                        "end": 1457
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 1480,
                                              "end": 1482
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1480,
                                            "end": 1482
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1503,
                                              "end": 1513
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1503,
                                            "end": 1513
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1534,
                                              "end": 1544
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1534,
                                            "end": 1544
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 1565,
                                              "end": 1569
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1565,
                                            "end": 1569
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 1590,
                                              "end": 1601
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1590,
                                            "end": 1601
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 1622,
                                              "end": 1629
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1622,
                                            "end": 1629
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 1650,
                                              "end": 1655
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1650,
                                            "end": 1655
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 1676,
                                              "end": 1686
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1676,
                                            "end": 1686
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 1707,
                                              "end": 1720
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 1747,
                                                    "end": 1749
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1747,
                                                  "end": 1749
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1774,
                                                    "end": 1784
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1774,
                                                  "end": 1784
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1809,
                                                    "end": 1819
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1809,
                                                  "end": 1819
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 1844,
                                                    "end": 1848
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1844,
                                                  "end": 1848
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 1873,
                                                    "end": 1884
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1873,
                                                  "end": 1884
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 1909,
                                                    "end": 1916
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1909,
                                                  "end": 1916
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 1941,
                                                    "end": 1946
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1941,
                                                  "end": 1946
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 1971,
                                                    "end": 1981
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1971,
                                                  "end": 1981
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1721,
                                              "end": 2003
                                            }
                                          },
                                          "loc": {
                                            "start": 1707,
                                            "end": 2003
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1458,
                                        "end": 2021
                                      }
                                    },
                                    "loc": {
                                      "start": 1448,
                                      "end": 2021
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 1357,
                                  "end": 2035
                                }
                              },
                              "loc": {
                                "start": 1344,
                                "end": 2035
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 2048,
                                  "end": 2060
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2079,
                                        "end": 2081
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2079,
                                      "end": 2081
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 2098,
                                        "end": 2108
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2098,
                                      "end": 2108
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 2125,
                                        "end": 2137
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2160,
                                              "end": 2162
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2160,
                                            "end": 2162
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 2183,
                                              "end": 2191
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2183,
                                            "end": 2191
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2212,
                                              "end": 2223
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2212,
                                            "end": 2223
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2244,
                                              "end": 2248
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2244,
                                            "end": 2248
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2138,
                                        "end": 2266
                                      }
                                    },
                                    "loc": {
                                      "start": 2125,
                                      "end": 2266
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 2283,
                                        "end": 2292
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2315,
                                              "end": 2317
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2315,
                                            "end": 2317
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 2338,
                                              "end": 2343
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2338,
                                            "end": 2343
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 2364,
                                              "end": 2368
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2364,
                                            "end": 2368
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 2389,
                                              "end": 2396
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2389,
                                            "end": 2396
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2417,
                                              "end": 2429
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "selectionSet": {
                                            "kind": "SelectionSet",
                                            "selections": [
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "id",
                                                  "loc": {
                                                    "start": 2456,
                                                    "end": 2458
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2456,
                                                  "end": 2458
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2483,
                                                    "end": 2491
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2483,
                                                  "end": 2491
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2516,
                                                    "end": 2527
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2516,
                                                  "end": 2527
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2552,
                                                    "end": 2556
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2552,
                                                  "end": 2556
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2430,
                                              "end": 2578
                                            }
                                          },
                                          "loc": {
                                            "start": 2417,
                                            "end": 2578
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2293,
                                        "end": 2596
                                      }
                                    },
                                    "loc": {
                                      "start": 2283,
                                      "end": 2596
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2061,
                                  "end": 2610
                                }
                              },
                              "loc": {
                                "start": 2048,
                                "end": 2610
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 2623,
                                  "end": 2631
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "FragmentSpread",
                                    "name": {
                                      "kind": "Name",
                                      "value": "Schedule_common",
                                      "loc": {
                                        "start": 2653,
                                        "end": 2668
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 2650,
                                      "end": 2668
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 2632,
                                  "end": 2682
                                }
                              },
                              "loc": {
                                "start": 2623,
                                "end": 2682
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2695,
                                  "end": 2697
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2695,
                                "end": 2697
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 2710,
                                  "end": 2714
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2710,
                                "end": 2714
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 2727,
                                  "end": 2738
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2727,
                                "end": 2738
                              }
                            }
                          ],
                          "loc": {
                            "start": 374,
                            "end": 2748
                          }
                        },
                        "loc": {
                          "start": 369,
                          "end": 2748
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 2757,
                            "end": 2770
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2757,
                          "end": 2770
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 2779,
                            "end": 2787
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2779,
                          "end": 2787
                        }
                      }
                    ],
                    "loc": {
                      "start": 359,
                      "end": 2793
                    }
                  },
                  "loc": {
                    "start": 343,
                    "end": 2793
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 2798,
                      "end": 2807
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 2798,
                    "end": 2807
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 2812,
                      "end": 2825
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 2836,
                            "end": 2838
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2836,
                          "end": 2838
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 2847,
                            "end": 2857
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2847,
                          "end": 2857
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 2866,
                            "end": 2876
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2866,
                          "end": 2876
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 2885,
                            "end": 2890
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2885,
                          "end": 2890
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 2899,
                            "end": 2913
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 2899,
                          "end": 2913
                        }
                      }
                    ],
                    "loc": {
                      "start": 2826,
                      "end": 2919
                    }
                  },
                  "loc": {
                    "start": 2812,
                    "end": 2919
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 2924,
                      "end": 2934
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "selectionSet": {
                    "kind": "SelectionSet",
                    "selections": [
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filters",
                          "loc": {
                            "start": 2945,
                            "end": 2952
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 2967,
                                  "end": 2969
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2967,
                                "end": 2969
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 2982,
                                  "end": 2992
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 2982,
                                "end": 2992
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 3005,
                                  "end": 3008
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3027,
                                        "end": 3029
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3027,
                                      "end": 3029
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3046,
                                        "end": 3056
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3046,
                                      "end": 3056
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 3073,
                                        "end": 3076
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3073,
                                      "end": 3076
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 3093,
                                        "end": 3102
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3093,
                                      "end": 3102
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3119,
                                        "end": 3131
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3154,
                                              "end": 3156
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3154,
                                            "end": 3156
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3177,
                                              "end": 3185
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3177,
                                            "end": 3185
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3206,
                                              "end": 3217
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3206,
                                            "end": 3217
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3132,
                                        "end": 3235
                                      }
                                    },
                                    "loc": {
                                      "start": 3119,
                                      "end": 3235
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 3252,
                                        "end": 3255
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isOwn",
                                            "loc": {
                                              "start": 3278,
                                              "end": 3283
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3278,
                                            "end": 3283
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 3304,
                                              "end": 3316
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3304,
                                            "end": 3316
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3256,
                                        "end": 3334
                                      }
                                    },
                                    "loc": {
                                      "start": 3252,
                                      "end": 3334
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3009,
                                  "end": 3348
                                }
                              },
                              "loc": {
                                "start": 3005,
                                "end": 3348
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 3361,
                                  "end": 3370
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "labels",
                                      "loc": {
                                        "start": 3389,
                                        "end": 3395
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3418,
                                              "end": 3420
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3418,
                                            "end": 3420
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 3441,
                                              "end": 3446
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3441,
                                            "end": 3446
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 3467,
                                              "end": 3472
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3467,
                                            "end": 3472
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3396,
                                        "end": 3490
                                      }
                                    },
                                    "loc": {
                                      "start": 3389,
                                      "end": 3490
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 3507,
                                        "end": 3515
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "FragmentSpread",
                                          "name": {
                                            "kind": "Name",
                                            "value": "Schedule_common",
                                            "loc": {
                                              "start": 3541,
                                              "end": 3556
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 3538,
                                            "end": 3556
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3516,
                                        "end": 3574
                                      }
                                    },
                                    "loc": {
                                      "start": 3507,
                                      "end": 3574
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3591,
                                        "end": 3593
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3591,
                                      "end": 3593
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3610,
                                        "end": 3614
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3610,
                                      "end": 3614
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3631,
                                        "end": 3642
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3631,
                                      "end": 3642
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3371,
                                  "end": 3656
                                }
                              },
                              "loc": {
                                "start": 3361,
                                "end": 3656
                              }
                            }
                          ],
                          "loc": {
                            "start": 2953,
                            "end": 3666
                          }
                        },
                        "loc": {
                          "start": 2945,
                          "end": 3666
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 3675,
                            "end": 3681
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3696,
                                  "end": 3698
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3696,
                                "end": 3698
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 3711,
                                  "end": 3716
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3711,
                                "end": 3716
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 3729,
                                  "end": 3734
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3729,
                                "end": 3734
                              }
                            }
                          ],
                          "loc": {
                            "start": 3682,
                            "end": 3744
                          }
                        },
                        "loc": {
                          "start": 3675,
                          "end": 3744
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 3753,
                            "end": 3765
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 3780,
                                  "end": 3782
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3780,
                                "end": 3782
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3795,
                                  "end": 3805
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3795,
                                "end": 3805
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3818,
                                  "end": 3828
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3818,
                                "end": 3828
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 3841,
                                  "end": 3850
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 3869,
                                        "end": 3871
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3869,
                                      "end": 3871
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3888,
                                        "end": 3898
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3888,
                                      "end": 3898
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3915,
                                        "end": 3925
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3915,
                                      "end": 3925
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3942,
                                        "end": 3946
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3942,
                                      "end": 3946
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3963,
                                        "end": 3974
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3963,
                                      "end": 3974
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 3991,
                                        "end": 3998
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3991,
                                      "end": 3998
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4015,
                                        "end": 4020
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4015,
                                      "end": 4020
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 4037,
                                        "end": 4047
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4037,
                                      "end": 4047
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 4064,
                                        "end": 4077
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4100,
                                              "end": 4102
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4100,
                                            "end": 4102
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 4123,
                                              "end": 4133
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4123,
                                            "end": 4133
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 4154,
                                              "end": 4164
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4154,
                                            "end": 4164
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4185,
                                              "end": 4189
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4185,
                                            "end": 4189
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4210,
                                              "end": 4221
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4210,
                                            "end": 4221
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 4242,
                                              "end": 4249
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4242,
                                            "end": 4249
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4270,
                                              "end": 4275
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4270,
                                            "end": 4275
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 4296,
                                              "end": 4306
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4296,
                                            "end": 4306
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4078,
                                        "end": 4324
                                      }
                                    },
                                    "loc": {
                                      "start": 4064,
                                      "end": 4324
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3851,
                                  "end": 4338
                                }
                              },
                              "loc": {
                                "start": 3841,
                                "end": 4338
                              }
                            }
                          ],
                          "loc": {
                            "start": 3766,
                            "end": 4348
                          }
                        },
                        "loc": {
                          "start": 3753,
                          "end": 4348
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 4357,
                            "end": 4369
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4384,
                                  "end": 4386
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4384,
                                "end": 4386
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4399,
                                  "end": 4409
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4399,
                                "end": 4409
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 4422,
                                  "end": 4434
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4453,
                                        "end": 4455
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4453,
                                      "end": 4455
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 4472,
                                        "end": 4480
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4472,
                                      "end": 4480
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4497,
                                        "end": 4508
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4497,
                                      "end": 4508
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4525,
                                        "end": 4529
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4525,
                                      "end": 4529
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4435,
                                  "end": 4543
                                }
                              },
                              "loc": {
                                "start": 4422,
                                "end": 4543
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 4556,
                                  "end": 4565
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "selectionSet": {
                                "kind": "SelectionSet",
                                "selections": [
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 4584,
                                        "end": 4586
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4584,
                                      "end": 4586
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4603,
                                        "end": 4608
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4603,
                                      "end": 4608
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 4625,
                                        "end": 4629
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4625,
                                      "end": 4629
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 4646,
                                        "end": 4653
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4646,
                                      "end": 4653
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4670,
                                        "end": 4682
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "selectionSet": {
                                      "kind": "SelectionSet",
                                      "selections": [
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 4705,
                                              "end": 4707
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4705,
                                            "end": 4707
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4728,
                                              "end": 4736
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4728,
                                            "end": 4736
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4757,
                                              "end": 4768
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4757,
                                            "end": 4768
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4789,
                                              "end": 4793
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4789,
                                            "end": 4793
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4683,
                                        "end": 4811
                                      }
                                    },
                                    "loc": {
                                      "start": 4670,
                                      "end": 4811
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4566,
                                  "end": 4825
                                }
                              },
                              "loc": {
                                "start": 4556,
                                "end": 4825
                              }
                            }
                          ],
                          "loc": {
                            "start": 4370,
                            "end": 4835
                          }
                        },
                        "loc": {
                          "start": 4357,
                          "end": 4835
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 4844,
                            "end": 4852
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "selectionSet": {
                          "kind": "SelectionSet",
                          "selections": [
                            {
                              "kind": "FragmentSpread",
                              "name": {
                                "kind": "Name",
                                "value": "Schedule_common",
                                "loc": {
                                  "start": 4870,
                                  "end": 4885
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 4867,
                                "end": 4885
                              }
                            }
                          ],
                          "loc": {
                            "start": 4853,
                            "end": 4895
                          }
                        },
                        "loc": {
                          "start": 4844,
                          "end": 4895
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4904,
                            "end": 4906
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4904,
                          "end": 4906
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4915,
                            "end": 4919
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4915,
                          "end": 4919
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4928,
                            "end": 4939
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4928,
                          "end": 4939
                        }
                      }
                    ],
                    "loc": {
                      "start": 2935,
                      "end": 4945
                    }
                  },
                  "loc": {
                    "start": 2924,
                    "end": 4945
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 4950,
                      "end": 4956
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4950,
                    "end": 4956
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 4961,
                      "end": 4971
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4961,
                    "end": 4971
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 4976,
                      "end": 4978
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4976,
                    "end": 4978
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 4983,
                      "end": 4992
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4983,
                    "end": 4992
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 4997,
                      "end": 5013
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4997,
                    "end": 5013
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 5018,
                      "end": 5022
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5018,
                    "end": 5022
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 5027,
                      "end": 5037
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5027,
                    "end": 5037
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 5042,
                      "end": 5055
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5042,
                    "end": 5055
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 5060,
                      "end": 5079
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5060,
                    "end": 5079
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 5084,
                      "end": 5097
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5084,
                    "end": 5097
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 5102,
                      "end": 5121
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5102,
                    "end": 5121
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 5126,
                      "end": 5140
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5126,
                    "end": 5140
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 5145,
                      "end": 5150
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5145,
                    "end": 5150
                  }
                }
              ],
              "loc": {
                "start": 337,
                "end": 5152
              }
            },
            "loc": {
              "start": 331,
              "end": 5152
            }
          }
        ],
        "loc": {
          "start": 309,
          "end": 5154
        }
      },
      "loc": {
        "start": 276,
        "end": 5154
      }
    },
    "Wallet_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Wallet_common",
        "loc": {
          "start": 5164,
          "end": 5177
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Wallet",
          "loc": {
            "start": 5181,
            "end": 5187
          }
        },
        "loc": {
          "start": 5181,
          "end": 5187
        }
      },
      "directives": [],
      "selectionSet": {
        "kind": "SelectionSet",
        "selections": [
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 5190,
                "end": 5192
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5190,
              "end": 5192
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handles",
              "loc": {
                "start": 5193,
                "end": 5200
              }
            },
            "arguments": [],
            "directives": [],
            "selectionSet": {
              "kind": "SelectionSet",
              "selections": [
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 5207,
                      "end": 5209
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5207,
                    "end": 5209
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 5214,
                      "end": 5220
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5214,
                    "end": 5220
                  }
                }
              ],
              "loc": {
                "start": 5201,
                "end": 5222
              }
            },
            "loc": {
              "start": 5193,
              "end": 5222
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 5223,
                "end": 5227
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5223,
              "end": 5227
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "publicAddress",
              "loc": {
                "start": 5228,
                "end": 5241
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5228,
              "end": 5241
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stakingAddress",
              "loc": {
                "start": 5242,
                "end": 5256
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5242,
              "end": 5256
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "verified",
              "loc": {
                "start": 5257,
                "end": 5265
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 5257,
              "end": 5265
            }
          }
        ],
        "loc": {
          "start": 5188,
          "end": 5267
        }
      },
      "loc": {
        "start": 5155,
        "end": 5267
      }
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "walletComplete",
      "loc": {
        "start": 5278,
        "end": 5292
      }
    },
    "variableDefinitions": [
      {
        "kind": "VariableDefinition",
        "variable": {
          "kind": "Variable",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 5294,
              "end": 5299
            }
          },
          "loc": {
            "start": 5293,
            "end": 5299
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "WalletCompleteInput",
              "loc": {
                "start": 5301,
                "end": 5320
              }
            },
            "loc": {
              "start": 5301,
              "end": 5320
            }
          },
          "loc": {
            "start": 5301,
            "end": 5321
          }
        },
        "directives": [],
        "loc": {
          "start": 5293,
          "end": 5321
        }
      }
    ],
    "directives": [],
    "selectionSet": {
      "kind": "SelectionSet",
      "selections": [
        {
          "kind": "Field",
          "name": {
            "kind": "Name",
            "value": "walletComplete",
            "loc": {
              "start": 5327,
              "end": 5341
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 5342,
                  "end": 5347
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 5350,
                    "end": 5355
                  }
                },
                "loc": {
                  "start": 5349,
                  "end": 5355
                }
              },
              "loc": {
                "start": 5342,
                "end": 5355
              }
            }
          ],
          "directives": [],
          "selectionSet": {
            "kind": "SelectionSet",
            "selections": [
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "firstLogIn",
                  "loc": {
                    "start": 5363,
                    "end": 5373
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 5363,
                  "end": 5373
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "session",
                  "loc": {
                    "start": 5378,
                    "end": 5385
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Session_full",
                        "loc": {
                          "start": 5399,
                          "end": 5411
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 5396,
                        "end": 5411
                      }
                    }
                  ],
                  "loc": {
                    "start": 5386,
                    "end": 5417
                  }
                },
                "loc": {
                  "start": 5378,
                  "end": 5417
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "wallet",
                  "loc": {
                    "start": 5422,
                    "end": 5428
                  }
                },
                "arguments": [],
                "directives": [],
                "selectionSet": {
                  "kind": "SelectionSet",
                  "selections": [
                    {
                      "kind": "FragmentSpread",
                      "name": {
                        "kind": "Name",
                        "value": "Wallet_common",
                        "loc": {
                          "start": 5442,
                          "end": 5455
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 5439,
                        "end": 5455
                      }
                    }
                  ],
                  "loc": {
                    "start": 5429,
                    "end": 5461
                  }
                },
                "loc": {
                  "start": 5422,
                  "end": 5461
                }
              }
            ],
            "loc": {
              "start": 5357,
              "end": 5465
            }
          },
          "loc": {
            "start": 5327,
            "end": 5465
          }
        }
      ],
      "loc": {
        "start": 5323,
        "end": 5467
      }
    },
    "loc": {
      "start": 5269,
      "end": 5467
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_walletComplete"
  }
} as const;
