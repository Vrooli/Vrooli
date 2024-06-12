export const auth_walletComplete = {
  "fieldName": "walletComplete",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "walletComplete",
        "loc": {
          "start": 8882,
          "end": 8896
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 8897,
              "end": 8902
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 8905,
                "end": 8910
              }
            },
            "loc": {
              "start": 8904,
              "end": 8910
            }
          },
          "loc": {
            "start": 8897,
            "end": 8910
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
                "start": 8918,
                "end": 8928
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8918,
              "end": 8928
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "session",
              "loc": {
                "start": 8933,
                "end": 8940
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
                      "start": 8954,
                      "end": 8966
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8951,
                    "end": 8966
                  }
                }
              ],
              "loc": {
                "start": 8941,
                "end": 8972
              }
            },
            "loc": {
              "start": 8933,
              "end": 8972
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "wallet",
              "loc": {
                "start": 8977,
                "end": 8983
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
                      "start": 8997,
                      "end": 9010
                    }
                  },
                  "directives": [],
                  "loc": {
                    "start": 8994,
                    "end": 9010
                  }
                }
              ],
              "loc": {
                "start": 8984,
                "end": 9016
              }
            },
            "loc": {
              "start": 8977,
              "end": 9016
            }
          }
        ],
        "loc": {
          "start": 8912,
          "end": 9020
        }
      },
      "loc": {
        "start": 8882,
        "end": 9020
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
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 1046,
                                        "end": 1058
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
                                              "start": 1085,
                                              "end": 1087
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1085,
                                            "end": 1087
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 1112,
                                              "end": 1122
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1112,
                                            "end": 1122
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 1147,
                                              "end": 1157
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1147,
                                            "end": 1157
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 1182,
                                              "end": 1191
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
                                                    "start": 1222,
                                                    "end": 1224
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1222,
                                                  "end": 1224
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1253,
                                                    "end": 1263
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1253,
                                                  "end": 1263
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1292,
                                                    "end": 1302
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1292,
                                                  "end": 1302
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 1331,
                                                    "end": 1335
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1331,
                                                  "end": 1335
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 1364,
                                                    "end": 1375
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1364,
                                                  "end": 1375
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 1404,
                                                    "end": 1411
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1404,
                                                  "end": 1411
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 1440,
                                                    "end": 1445
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1440,
                                                  "end": 1445
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 1474,
                                                    "end": 1484
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1474,
                                                  "end": 1484
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 1513,
                                                    "end": 1526
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
                                                          "start": 1561,
                                                          "end": 1563
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1561,
                                                        "end": 1563
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1596,
                                                          "end": 1606
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1596,
                                                        "end": 1606
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1639,
                                                          "end": 1649
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1639,
                                                        "end": 1649
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1682,
                                                          "end": 1686
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1682,
                                                        "end": 1686
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1719,
                                                          "end": 1730
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1719,
                                                        "end": 1730
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1763,
                                                          "end": 1770
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1763,
                                                        "end": 1770
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1803,
                                                          "end": 1808
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1803,
                                                        "end": 1808
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1841,
                                                          "end": 1851
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1841,
                                                        "end": 1851
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1527,
                                                    "end": 1881
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1513,
                                                  "end": 1881
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1192,
                                              "end": 1907
                                            }
                                          },
                                          "loc": {
                                            "start": 1182,
                                            "end": 1907
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1059,
                                        "end": 1929
                                      }
                                    },
                                    "loc": {
                                      "start": 1046,
                                      "end": 1929
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 1950,
                                        "end": 1962
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
                                              "start": 1989,
                                              "end": 1991
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 1989,
                                            "end": 1991
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 2016,
                                              "end": 2026
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2016,
                                            "end": 2026
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 2051,
                                              "end": 2063
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
                                                    "start": 2094,
                                                    "end": 2096
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2094,
                                                  "end": 2096
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 2125,
                                                    "end": 2133
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2125,
                                                  "end": 2133
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 2162,
                                                    "end": 2173
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2162,
                                                  "end": 2173
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 2202,
                                                    "end": 2206
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2202,
                                                  "end": 2206
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2064,
                                              "end": 2232
                                            }
                                          },
                                          "loc": {
                                            "start": 2051,
                                            "end": 2232
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 2257,
                                              "end": 2266
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
                                                    "start": 2297,
                                                    "end": 2299
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2297,
                                                  "end": 2299
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 2328,
                                                    "end": 2333
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2328,
                                                  "end": 2333
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 2362,
                                                    "end": 2366
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2362,
                                                  "end": 2366
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 2395,
                                                    "end": 2402
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2395,
                                                  "end": 2402
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2431,
                                                    "end": 2443
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
                                                          "start": 2478,
                                                          "end": 2480
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2478,
                                                        "end": 2480
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2513,
                                                          "end": 2521
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2513,
                                                        "end": 2521
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2554,
                                                          "end": 2565
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2554,
                                                        "end": 2565
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2598,
                                                          "end": 2602
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2598,
                                                        "end": 2602
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2444,
                                                    "end": 2632
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2431,
                                                  "end": 2632
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2267,
                                              "end": 2658
                                            }
                                          },
                                          "loc": {
                                            "start": 2257,
                                            "end": 2658
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1963,
                                        "end": 2680
                                      }
                                    },
                                    "loc": {
                                      "start": 1950,
                                      "end": 2680
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 2701,
                                        "end": 2709
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
                                              "start": 2739,
                                              "end": 2754
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 2736,
                                            "end": 2754
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2710,
                                        "end": 2776
                                      }
                                    },
                                    "loc": {
                                      "start": 2701,
                                      "end": 2776
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 2797,
                                        "end": 2799
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2797,
                                      "end": 2799
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 2820,
                                        "end": 2824
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2820,
                                      "end": 2824
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 2845,
                                        "end": 2856
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 2845,
                                      "end": 2856
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 2877,
                                        "end": 2880
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
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 2907,
                                              "end": 2916
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2907,
                                            "end": 2916
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 2941,
                                              "end": 2948
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2941,
                                            "end": 2948
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 2973,
                                              "end": 2982
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2973,
                                            "end": 2982
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 2881,
                                        "end": 3004
                                      }
                                    },
                                    "loc": {
                                      "start": 2877,
                                      "end": 3004
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 886,
                                  "end": 3022
                                }
                              },
                              "loc": {
                                "start": 876,
                                "end": 3022
                              }
                            }
                          ],
                          "loc": {
                            "start": 396,
                            "end": 3036
                          }
                        },
                        "loc": {
                          "start": 388,
                          "end": 3036
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 3049,
                            "end": 3055
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
                                  "start": 3074,
                                  "end": 3076
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3074,
                                "end": 3076
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 3093,
                                  "end": 3098
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3093,
                                "end": 3098
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 3115,
                                  "end": 3120
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3115,
                                "end": 3120
                              }
                            }
                          ],
                          "loc": {
                            "start": 3056,
                            "end": 3134
                          }
                        },
                        "loc": {
                          "start": 3049,
                          "end": 3134
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 3147,
                            "end": 3159
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
                                  "start": 3178,
                                  "end": 3180
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3178,
                                "end": 3180
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3197,
                                  "end": 3207
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3197,
                                "end": 3207
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 3224,
                                  "end": 3234
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3224,
                                "end": 3234
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 3251,
                                  "end": 3260
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
                                        "start": 3283,
                                        "end": 3285
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3283,
                                      "end": 3285
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3306,
                                        "end": 3316
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3306,
                                      "end": 3316
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3337,
                                        "end": 3347
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3337,
                                      "end": 3347
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 3368,
                                        "end": 3372
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3368,
                                      "end": 3372
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 3393,
                                        "end": 3404
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3393,
                                      "end": 3404
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 3425,
                                        "end": 3432
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3425,
                                      "end": 3432
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 3453,
                                        "end": 3458
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3453,
                                      "end": 3458
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 3479,
                                        "end": 3489
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3479,
                                      "end": 3489
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 3510,
                                        "end": 3523
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
                                              "start": 3550,
                                              "end": 3552
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3550,
                                            "end": 3552
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3577,
                                              "end": 3587
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3577,
                                            "end": 3587
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3612,
                                              "end": 3622
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3612,
                                            "end": 3622
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3647,
                                              "end": 3651
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3647,
                                            "end": 3651
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3676,
                                              "end": 3687
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3676,
                                            "end": 3687
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3712,
                                              "end": 3719
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3712,
                                            "end": 3719
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3744,
                                              "end": 3749
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3744,
                                            "end": 3749
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3774,
                                              "end": 3784
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3774,
                                            "end": 3784
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3524,
                                        "end": 3806
                                      }
                                    },
                                    "loc": {
                                      "start": 3510,
                                      "end": 3806
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3261,
                                  "end": 3824
                                }
                              },
                              "loc": {
                                "start": 3251,
                                "end": 3824
                              }
                            }
                          ],
                          "loc": {
                            "start": 3160,
                            "end": 3838
                          }
                        },
                        "loc": {
                          "start": 3147,
                          "end": 3838
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 3851,
                            "end": 3863
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
                                  "start": 3882,
                                  "end": 3884
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3882,
                                "end": 3884
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 3901,
                                  "end": 3911
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 3901,
                                "end": 3911
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 3928,
                                  "end": 3940
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
                                        "start": 3963,
                                        "end": 3965
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3963,
                                      "end": 3965
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 3986,
                                        "end": 3994
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3986,
                                      "end": 3994
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 4015,
                                        "end": 4026
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4015,
                                      "end": 4026
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 4047,
                                        "end": 4051
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4047,
                                      "end": 4051
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3941,
                                  "end": 4069
                                }
                              },
                              "loc": {
                                "start": 3928,
                                "end": 4069
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 4086,
                                  "end": 4095
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
                                        "start": 4118,
                                        "end": 4120
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4118,
                                      "end": 4120
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 4141,
                                        "end": 4146
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4141,
                                      "end": 4146
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 4167,
                                        "end": 4171
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4167,
                                      "end": 4171
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 4192,
                                        "end": 4199
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4192,
                                      "end": 4199
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4220,
                                        "end": 4232
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
                                              "start": 4259,
                                              "end": 4261
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4259,
                                            "end": 4261
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4286,
                                              "end": 4294
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4286,
                                            "end": 4294
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4319,
                                              "end": 4330
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4319,
                                            "end": 4330
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4355,
                                              "end": 4359
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4355,
                                            "end": 4359
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4233,
                                        "end": 4381
                                      }
                                    },
                                    "loc": {
                                      "start": 4220,
                                      "end": 4381
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4096,
                                  "end": 4399
                                }
                              },
                              "loc": {
                                "start": 4086,
                                "end": 4399
                              }
                            }
                          ],
                          "loc": {
                            "start": 3864,
                            "end": 4413
                          }
                        },
                        "loc": {
                          "start": 3851,
                          "end": 4413
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 4426,
                            "end": 4434
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
                                  "start": 4456,
                                  "end": 4471
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 4453,
                                "end": 4471
                              }
                            }
                          ],
                          "loc": {
                            "start": 4435,
                            "end": 4485
                          }
                        },
                        "loc": {
                          "start": 4426,
                          "end": 4485
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 4498,
                            "end": 4500
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4498,
                          "end": 4500
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 4513,
                            "end": 4517
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4513,
                          "end": 4517
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 4530,
                            "end": 4541
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4530,
                          "end": 4541
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 4554,
                            "end": 4557
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
                                "value": "canDelete",
                                "loc": {
                                  "start": 4576,
                                  "end": 4585
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4576,
                                "end": 4585
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 4602,
                                  "end": 4609
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4602,
                                "end": 4609
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 4626,
                                  "end": 4635
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4626,
                                "end": 4635
                              }
                            }
                          ],
                          "loc": {
                            "start": 4558,
                            "end": 4649
                          }
                        },
                        "loc": {
                          "start": 4554,
                          "end": 4649
                        }
                      }
                    ],
                    "loc": {
                      "start": 374,
                      "end": 4659
                    }
                  },
                  "loc": {
                    "start": 369,
                    "end": 4659
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stopCondition",
                    "loc": {
                      "start": 4668,
                      "end": 4681
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4668,
                    "end": 4681
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "stopTime",
                    "loc": {
                      "start": 4690,
                      "end": 4698
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4690,
                    "end": 4698
                  }
                }
              ],
              "loc": {
                "start": 359,
                "end": 4704
              }
            },
            "loc": {
              "start": 343,
              "end": 4704
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "apisCount",
              "loc": {
                "start": 4709,
                "end": 4718
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4709,
              "end": 4718
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "bookmarkLists",
              "loc": {
                "start": 4723,
                "end": 4736
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
                      "start": 4747,
                      "end": 4749
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4747,
                    "end": 4749
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "created_at",
                    "loc": {
                      "start": 4758,
                      "end": 4768
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4758,
                    "end": 4768
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "updated_at",
                    "loc": {
                      "start": 4777,
                      "end": 4787
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4777,
                    "end": 4787
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "label",
                    "loc": {
                      "start": 4796,
                      "end": 4801
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4796,
                    "end": 4801
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarksCount",
                    "loc": {
                      "start": 4810,
                      "end": 4824
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4810,
                    "end": 4824
                  }
                }
              ],
              "loc": {
                "start": 4737,
                "end": 4830
              }
            },
            "loc": {
              "start": 4723,
              "end": 4830
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "codesCount",
              "loc": {
                "start": 4835,
                "end": 4845
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 4835,
              "end": 4845
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "focusModes",
              "loc": {
                "start": 4850,
                "end": 4860
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
                      "start": 4871,
                      "end": 4878
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
                            "start": 4893,
                            "end": 4895
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4893,
                          "end": 4895
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "filterType",
                          "loc": {
                            "start": 4908,
                            "end": 4918
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4908,
                          "end": 4918
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "tag",
                          "loc": {
                            "start": 4931,
                            "end": 4934
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
                                  "start": 4953,
                                  "end": 4955
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4953,
                                "end": 4955
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 4972,
                                  "end": 4982
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4972,
                                "end": 4982
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 4999,
                                  "end": 5002
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4999,
                                "end": 5002
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "bookmarks",
                                "loc": {
                                  "start": 5019,
                                  "end": 5028
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5019,
                                "end": 5028
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 5045,
                                  "end": 5057
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
                                        "start": 5080,
                                        "end": 5082
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5080,
                                      "end": 5082
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 5103,
                                        "end": 5111
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5103,
                                      "end": 5111
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 5132,
                                        "end": 5143
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5132,
                                      "end": 5143
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5058,
                                  "end": 5161
                                }
                              },
                              "loc": {
                                "start": 5045,
                                "end": 5161
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 5178,
                                  "end": 5181
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
                                        "start": 5204,
                                        "end": 5209
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5204,
                                      "end": 5209
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isBookmarked",
                                      "loc": {
                                        "start": 5230,
                                        "end": 5242
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5230,
                                      "end": 5242
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5182,
                                  "end": 5260
                                }
                              },
                              "loc": {
                                "start": 5178,
                                "end": 5260
                              }
                            }
                          ],
                          "loc": {
                            "start": 4935,
                            "end": 5274
                          }
                        },
                        "loc": {
                          "start": 4931,
                          "end": 5274
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "focusMode",
                          "loc": {
                            "start": 5287,
                            "end": 5296
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
                                  "start": 5315,
                                  "end": 5321
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
                                        "start": 5344,
                                        "end": 5346
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5344,
                                      "end": 5346
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 5367,
                                        "end": 5372
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5367,
                                      "end": 5372
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 5393,
                                        "end": 5398
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5393,
                                      "end": 5398
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5322,
                                  "end": 5416
                                }
                              },
                              "loc": {
                                "start": 5315,
                                "end": 5416
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 5433,
                                  "end": 5445
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
                                        "start": 5468,
                                        "end": 5470
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5468,
                                      "end": 5470
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5491,
                                        "end": 5501
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5491,
                                      "end": 5501
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 5522,
                                        "end": 5532
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5522,
                                      "end": 5532
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 5553,
                                        "end": 5562
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
                                              "start": 5589,
                                              "end": 5591
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5589,
                                            "end": 5591
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5616,
                                              "end": 5626
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5616,
                                            "end": 5626
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 5651,
                                              "end": 5661
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5651,
                                            "end": 5661
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 5686,
                                              "end": 5690
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5686,
                                            "end": 5690
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5715,
                                              "end": 5726
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5715,
                                            "end": 5726
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 5751,
                                              "end": 5758
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5751,
                                            "end": 5758
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 5783,
                                              "end": 5788
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5783,
                                            "end": 5788
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 5813,
                                              "end": 5823
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5813,
                                            "end": 5823
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 5848,
                                              "end": 5861
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
                                                    "start": 5892,
                                                    "end": 5894
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5892,
                                                  "end": 5894
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 5923,
                                                    "end": 5933
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5923,
                                                  "end": 5933
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 5962,
                                                    "end": 5972
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5962,
                                                  "end": 5972
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6001,
                                                    "end": 6005
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6001,
                                                  "end": 6005
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6034,
                                                    "end": 6045
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6034,
                                                  "end": 6045
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 6074,
                                                    "end": 6081
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6074,
                                                  "end": 6081
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6110,
                                                    "end": 6115
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6110,
                                                  "end": 6115
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 6144,
                                                    "end": 6154
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6144,
                                                  "end": 6154
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5862,
                                              "end": 6180
                                            }
                                          },
                                          "loc": {
                                            "start": 5848,
                                            "end": 6180
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5563,
                                        "end": 6202
                                      }
                                    },
                                    "loc": {
                                      "start": 5553,
                                      "end": 6202
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5446,
                                  "end": 6220
                                }
                              },
                              "loc": {
                                "start": 5433,
                                "end": 6220
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 6237,
                                  "end": 6249
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
                                        "start": 6272,
                                        "end": 6274
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6272,
                                      "end": 6274
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 6295,
                                        "end": 6305
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6295,
                                      "end": 6305
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 6326,
                                        "end": 6338
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
                                              "start": 6365,
                                              "end": 6367
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6365,
                                            "end": 6367
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 6392,
                                              "end": 6400
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6392,
                                            "end": 6400
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 6425,
                                              "end": 6436
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6425,
                                            "end": 6436
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 6461,
                                              "end": 6465
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6461,
                                            "end": 6465
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6339,
                                        "end": 6487
                                      }
                                    },
                                    "loc": {
                                      "start": 6326,
                                      "end": 6487
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 6508,
                                        "end": 6517
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
                                              "start": 6544,
                                              "end": 6546
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6544,
                                            "end": 6546
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 6571,
                                              "end": 6576
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6571,
                                            "end": 6576
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 6601,
                                              "end": 6605
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6601,
                                            "end": 6605
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 6630,
                                              "end": 6637
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6630,
                                            "end": 6637
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6662,
                                              "end": 6674
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
                                                    "start": 6705,
                                                    "end": 6707
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6705,
                                                  "end": 6707
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 6736,
                                                    "end": 6744
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6736,
                                                  "end": 6744
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6773,
                                                    "end": 6784
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6773,
                                                  "end": 6784
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6813,
                                                    "end": 6817
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6813,
                                                  "end": 6817
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6675,
                                              "end": 6843
                                            }
                                          },
                                          "loc": {
                                            "start": 6662,
                                            "end": 6843
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6518,
                                        "end": 6865
                                      }
                                    },
                                    "loc": {
                                      "start": 6508,
                                      "end": 6865
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6250,
                                  "end": 6883
                                }
                              },
                              "loc": {
                                "start": 6237,
                                "end": 6883
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 6900,
                                  "end": 6908
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
                                        "start": 6934,
                                        "end": 6949
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 6931,
                                      "end": 6949
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 6909,
                                  "end": 6967
                                }
                              },
                              "loc": {
                                "start": 6900,
                                "end": 6967
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 6984,
                                  "end": 6986
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 6984,
                                "end": 6986
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7003,
                                  "end": 7007
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7003,
                                "end": 7007
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7024,
                                  "end": 7035
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7024,
                                "end": 7035
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 7052,
                                  "end": 7055
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
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 7078,
                                        "end": 7087
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7078,
                                      "end": 7087
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 7108,
                                        "end": 7115
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7108,
                                      "end": 7115
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 7136,
                                        "end": 7145
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7136,
                                      "end": 7145
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7056,
                                  "end": 7163
                                }
                              },
                              "loc": {
                                "start": 7052,
                                "end": 7163
                              }
                            }
                          ],
                          "loc": {
                            "start": 5297,
                            "end": 7177
                          }
                        },
                        "loc": {
                          "start": 5287,
                          "end": 7177
                        }
                      }
                    ],
                    "loc": {
                      "start": 4879,
                      "end": 7187
                    }
                  },
                  "loc": {
                    "start": 4871,
                    "end": 7187
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "labels",
                    "loc": {
                      "start": 7196,
                      "end": 7202
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
                            "start": 7217,
                            "end": 7219
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7217,
                          "end": 7219
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "color",
                          "loc": {
                            "start": 7232,
                            "end": 7237
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7232,
                          "end": 7237
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 7250,
                            "end": 7255
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7250,
                          "end": 7255
                        }
                      }
                    ],
                    "loc": {
                      "start": 7203,
                      "end": 7265
                    }
                  },
                  "loc": {
                    "start": 7196,
                    "end": 7265
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "reminderList",
                    "loc": {
                      "start": 7274,
                      "end": 7286
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
                            "start": 7301,
                            "end": 7303
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7301,
                          "end": 7303
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7316,
                            "end": 7326
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7316,
                          "end": 7326
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 7339,
                            "end": 7349
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7339,
                          "end": 7349
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminders",
                          "loc": {
                            "start": 7362,
                            "end": 7371
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
                                  "start": 7390,
                                  "end": 7392
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7390,
                                "end": 7392
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7409,
                                  "end": 7419
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7409,
                                "end": 7419
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7436,
                                  "end": 7446
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7436,
                                "end": 7446
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 7463,
                                  "end": 7467
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7463,
                                "end": 7467
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 7484,
                                  "end": 7495
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7484,
                                "end": 7495
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "dueDate",
                                "loc": {
                                  "start": 7512,
                                  "end": 7519
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7512,
                                "end": 7519
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 7536,
                                  "end": 7541
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7536,
                                "end": 7541
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "isComplete",
                                "loc": {
                                  "start": 7558,
                                  "end": 7568
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7558,
                                "end": 7568
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderItems",
                                "loc": {
                                  "start": 7585,
                                  "end": 7598
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
                                        "start": 7621,
                                        "end": 7623
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7621,
                                      "end": 7623
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 7644,
                                        "end": 7654
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7644,
                                      "end": 7654
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 7675,
                                        "end": 7685
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7675,
                                      "end": 7685
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7706,
                                        "end": 7710
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7706,
                                      "end": 7710
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7731,
                                        "end": 7742
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7731,
                                      "end": 7742
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 7763,
                                        "end": 7770
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7763,
                                      "end": 7770
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 7791,
                                        "end": 7796
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7791,
                                      "end": 7796
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 7817,
                                        "end": 7827
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7817,
                                      "end": 7827
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7599,
                                  "end": 7845
                                }
                              },
                              "loc": {
                                "start": 7585,
                                "end": 7845
                              }
                            }
                          ],
                          "loc": {
                            "start": 7372,
                            "end": 7859
                          }
                        },
                        "loc": {
                          "start": 7362,
                          "end": 7859
                        }
                      }
                    ],
                    "loc": {
                      "start": 7287,
                      "end": 7869
                    }
                  },
                  "loc": {
                    "start": 7274,
                    "end": 7869
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "resourceList",
                    "loc": {
                      "start": 7878,
                      "end": 7890
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
                            "start": 7905,
                            "end": 7907
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7905,
                          "end": 7907
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 7920,
                            "end": 7930
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 7920,
                          "end": 7930
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "translations",
                          "loc": {
                            "start": 7943,
                            "end": 7955
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
                                  "start": 7974,
                                  "end": 7976
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7974,
                                "end": 7976
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "language",
                                "loc": {
                                  "start": 7993,
                                  "end": 8001
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7993,
                                "end": 8001
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 8018,
                                  "end": 8029
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8018,
                                "end": 8029
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 8046,
                                  "end": 8050
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8046,
                                "end": 8050
                              }
                            }
                          ],
                          "loc": {
                            "start": 7956,
                            "end": 8064
                          }
                        },
                        "loc": {
                          "start": 7943,
                          "end": 8064
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resources",
                          "loc": {
                            "start": 8077,
                            "end": 8086
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
                                  "start": 8105,
                                  "end": 8107
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8105,
                                "end": 8107
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "index",
                                "loc": {
                                  "start": 8124,
                                  "end": 8129
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8124,
                                "end": 8129
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "link",
                                "loc": {
                                  "start": 8146,
                                  "end": 8150
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8146,
                                "end": 8150
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "usedFor",
                                "loc": {
                                  "start": 8167,
                                  "end": 8174
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8167,
                                "end": 8174
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 8191,
                                  "end": 8203
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
                                        "start": 8226,
                                        "end": 8228
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8226,
                                      "end": 8228
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 8249,
                                        "end": 8257
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8249,
                                      "end": 8257
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8278,
                                        "end": 8289
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8278,
                                      "end": 8289
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8310,
                                        "end": 8314
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8310,
                                      "end": 8314
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8204,
                                  "end": 8332
                                }
                              },
                              "loc": {
                                "start": 8191,
                                "end": 8332
                              }
                            }
                          ],
                          "loc": {
                            "start": 8087,
                            "end": 8346
                          }
                        },
                        "loc": {
                          "start": 8077,
                          "end": 8346
                        }
                      }
                    ],
                    "loc": {
                      "start": 7891,
                      "end": 8356
                    }
                  },
                  "loc": {
                    "start": 7878,
                    "end": 8356
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "schedule",
                    "loc": {
                      "start": 8365,
                      "end": 8373
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
                            "start": 8391,
                            "end": 8406
                          }
                        },
                        "directives": [],
                        "loc": {
                          "start": 8388,
                          "end": 8406
                        }
                      }
                    ],
                    "loc": {
                      "start": 8374,
                      "end": 8416
                    }
                  },
                  "loc": {
                    "start": 8365,
                    "end": 8416
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8425,
                      "end": 8427
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8425,
                    "end": 8427
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 8436,
                      "end": 8440
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8436,
                    "end": 8440
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "description",
                    "loc": {
                      "start": 8449,
                      "end": 8460
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8449,
                    "end": 8460
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "you",
                    "loc": {
                      "start": 8469,
                      "end": 8472
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
                          "value": "canDelete",
                          "loc": {
                            "start": 8487,
                            "end": 8496
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8487,
                          "end": 8496
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canRead",
                          "loc": {
                            "start": 8509,
                            "end": 8516
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8509,
                          "end": 8516
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "canUpdate",
                          "loc": {
                            "start": 8529,
                            "end": 8538
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8529,
                          "end": 8538
                        }
                      }
                    ],
                    "loc": {
                      "start": 8473,
                      "end": 8548
                    }
                  },
                  "loc": {
                    "start": 8469,
                    "end": 8548
                  }
                }
              ],
              "loc": {
                "start": 4861,
                "end": 8554
              }
            },
            "loc": {
              "start": 4850,
              "end": 8554
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "handle",
              "loc": {
                "start": 8559,
                "end": 8565
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8559,
              "end": 8565
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "hasPremium",
              "loc": {
                "start": 8570,
                "end": 8580
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8570,
              "end": 8580
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "id",
              "loc": {
                "start": 8585,
                "end": 8587
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8585,
              "end": 8587
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "languages",
              "loc": {
                "start": 8592,
                "end": 8601
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8592,
              "end": 8601
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "membershipsCount",
              "loc": {
                "start": 8606,
                "end": 8622
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8606,
              "end": 8622
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8627,
                "end": 8631
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8627,
              "end": 8631
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "notesCount",
              "loc": {
                "start": 8636,
                "end": 8646
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8636,
              "end": 8646
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "projectsCount",
              "loc": {
                "start": 8651,
                "end": 8664
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8651,
              "end": 8664
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "questionsAskedCount",
              "loc": {
                "start": 8669,
                "end": 8688
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8669,
              "end": 8688
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "routinesCount",
              "loc": {
                "start": 8693,
                "end": 8706
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8693,
              "end": 8706
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "standardsCount",
              "loc": {
                "start": 8711,
                "end": 8725
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8711,
              "end": 8725
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "theme",
              "loc": {
                "start": 8730,
                "end": 8735
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8730,
              "end": 8735
            }
          }
        ],
        "loc": {
          "start": 337,
          "end": 8737
        }
      },
      "loc": {
        "start": 331,
        "end": 8737
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "id",
        "loc": {
          "start": 8775,
          "end": 8777
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8775,
        "end": 8777
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "name",
        "loc": {
          "start": 8778,
          "end": 8782
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8778,
        "end": 8782
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "publicAddress",
        "loc": {
          "start": 8783,
          "end": 8796
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8783,
        "end": 8796
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "stakingAddress",
        "loc": {
          "start": 8797,
          "end": 8811
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8797,
        "end": 8811
      }
    },
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "verified",
        "loc": {
          "start": 8812,
          "end": 8820
        }
      },
      "arguments": [],
      "directives": [],
      "loc": {
        "start": 8812,
        "end": 8820
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
                                            "value": "reminderList",
                                            "loc": {
                                              "start": 1046,
                                              "end": 1058
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
                                                    "start": 1085,
                                                    "end": 1087
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1085,
                                                  "end": 1087
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1112,
                                                    "end": 1122
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1112,
                                                  "end": 1122
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1147,
                                                    "end": 1157
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1147,
                                                  "end": 1157
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminders",
                                                  "loc": {
                                                    "start": 1182,
                                                    "end": 1191
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
                                                          "start": 1222,
                                                          "end": 1224
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1222,
                                                        "end": 1224
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1253,
                                                          "end": 1263
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1253,
                                                        "end": 1263
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1292,
                                                          "end": 1302
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1292,
                                                        "end": 1302
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1331,
                                                          "end": 1335
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1331,
                                                        "end": 1335
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1364,
                                                          "end": 1375
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1364,
                                                        "end": 1375
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1404,
                                                          "end": 1411
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1404,
                                                        "end": 1411
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1440,
                                                          "end": 1445
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1440,
                                                        "end": 1445
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1474,
                                                          "end": 1484
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1474,
                                                        "end": 1484
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "reminderItems",
                                                        "loc": {
                                                          "start": 1513,
                                                          "end": 1526
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
                                                                "start": 1561,
                                                                "end": 1563
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1561,
                                                              "end": 1563
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "created_at",
                                                              "loc": {
                                                                "start": 1596,
                                                                "end": 1606
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1596,
                                                              "end": 1606
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "updated_at",
                                                              "loc": {
                                                                "start": 1639,
                                                                "end": 1649
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1639,
                                                              "end": 1649
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 1682,
                                                                "end": 1686
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1682,
                                                              "end": 1686
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 1719,
                                                                "end": 1730
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1719,
                                                              "end": 1730
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "dueDate",
                                                              "loc": {
                                                                "start": 1763,
                                                                "end": 1770
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1763,
                                                              "end": 1770
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "index",
                                                              "loc": {
                                                                "start": 1803,
                                                                "end": 1808
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1803,
                                                              "end": 1808
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isComplete",
                                                              "loc": {
                                                                "start": 1841,
                                                                "end": 1851
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1841,
                                                              "end": 1851
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1527,
                                                          "end": 1881
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1513,
                                                        "end": 1881
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1192,
                                                    "end": 1907
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1182,
                                                  "end": 1907
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1059,
                                              "end": 1929
                                            }
                                          },
                                          "loc": {
                                            "start": 1046,
                                            "end": 1929
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resourceList",
                                            "loc": {
                                              "start": 1950,
                                              "end": 1962
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
                                                    "start": 1989,
                                                    "end": 1991
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1989,
                                                  "end": 1991
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2016,
                                                    "end": 2026
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2016,
                                                  "end": 2026
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2051,
                                                    "end": 2063
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
                                                          "start": 2094,
                                                          "end": 2096
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2094,
                                                        "end": 2096
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2125,
                                                          "end": 2133
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2125,
                                                        "end": 2133
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2162,
                                                          "end": 2173
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2162,
                                                        "end": 2173
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2202,
                                                          "end": 2206
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2202,
                                                        "end": 2206
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2064,
                                                    "end": 2232
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2051,
                                                  "end": 2232
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "resources",
                                                  "loc": {
                                                    "start": 2257,
                                                    "end": 2266
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
                                                          "start": 2297,
                                                          "end": 2299
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2297,
                                                        "end": 2299
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 2328,
                                                          "end": 2333
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2328,
                                                        "end": 2333
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "link",
                                                        "loc": {
                                                          "start": 2362,
                                                          "end": 2366
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2362,
                                                        "end": 2366
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "usedFor",
                                                        "loc": {
                                                          "start": 2395,
                                                          "end": 2402
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2395,
                                                        "end": 2402
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "translations",
                                                        "loc": {
                                                          "start": 2431,
                                                          "end": 2443
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
                                                                "start": 2478,
                                                                "end": 2480
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2478,
                                                              "end": 2480
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "language",
                                                              "loc": {
                                                                "start": 2513,
                                                                "end": 2521
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2513,
                                                              "end": 2521
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 2554,
                                                                "end": 2565
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2554,
                                                              "end": 2565
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 2598,
                                                                "end": 2602
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2598,
                                                              "end": 2602
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 2444,
                                                          "end": 2632
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2431,
                                                        "end": 2632
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2267,
                                                    "end": 2658
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2257,
                                                  "end": 2658
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1963,
                                              "end": 2680
                                            }
                                          },
                                          "loc": {
                                            "start": 1950,
                                            "end": 2680
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 2701,
                                              "end": 2709
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
                                                    "start": 2739,
                                                    "end": 2754
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 2736,
                                                  "end": 2754
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2710,
                                              "end": 2776
                                            }
                                          },
                                          "loc": {
                                            "start": 2701,
                                            "end": 2776
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 2797,
                                              "end": 2799
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2797,
                                            "end": 2799
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 2820,
                                              "end": 2824
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2820,
                                            "end": 2824
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 2845,
                                              "end": 2856
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 2845,
                                            "end": 2856
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 2877,
                                              "end": 2880
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
                                                  "value": "canDelete",
                                                  "loc": {
                                                    "start": 2907,
                                                    "end": 2916
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2907,
                                                  "end": 2916
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canRead",
                                                  "loc": {
                                                    "start": 2941,
                                                    "end": 2948
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2941,
                                                  "end": 2948
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "canUpdate",
                                                  "loc": {
                                                    "start": 2973,
                                                    "end": 2982
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2973,
                                                  "end": 2982
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2881,
                                              "end": 3004
                                            }
                                          },
                                          "loc": {
                                            "start": 2877,
                                            "end": 3004
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 886,
                                        "end": 3022
                                      }
                                    },
                                    "loc": {
                                      "start": 876,
                                      "end": 3022
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 396,
                                  "end": 3036
                                }
                              },
                              "loc": {
                                "start": 388,
                                "end": 3036
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 3049,
                                  "end": 3055
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
                                        "start": 3074,
                                        "end": 3076
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3074,
                                      "end": 3076
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 3093,
                                        "end": 3098
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3093,
                                      "end": 3098
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 3115,
                                        "end": 3120
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3115,
                                      "end": 3120
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3056,
                                  "end": 3134
                                }
                              },
                              "loc": {
                                "start": 3049,
                                "end": 3134
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 3147,
                                  "end": 3159
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
                                        "start": 3178,
                                        "end": 3180
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3178,
                                      "end": 3180
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3197,
                                        "end": 3207
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3197,
                                      "end": 3207
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3224,
                                        "end": 3234
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3224,
                                      "end": 3234
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 3251,
                                        "end": 3260
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
                                              "start": 3283,
                                              "end": 3285
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3283,
                                            "end": 3285
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3306,
                                              "end": 3316
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3306,
                                            "end": 3316
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3337,
                                              "end": 3347
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3337,
                                            "end": 3347
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3368,
                                              "end": 3372
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3368,
                                            "end": 3372
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3393,
                                              "end": 3404
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3393,
                                            "end": 3404
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3425,
                                              "end": 3432
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3425,
                                            "end": 3432
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3453,
                                              "end": 3458
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3453,
                                            "end": 3458
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3479,
                                              "end": 3489
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3479,
                                            "end": 3489
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 3510,
                                              "end": 3523
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
                                                    "start": 3550,
                                                    "end": 3552
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3550,
                                                  "end": 3552
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3577,
                                                    "end": 3587
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3577,
                                                  "end": 3587
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3612,
                                                    "end": 3622
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3612,
                                                  "end": 3622
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 3647,
                                                    "end": 3651
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3647,
                                                  "end": 3651
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 3676,
                                                    "end": 3687
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3676,
                                                  "end": 3687
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 3712,
                                                    "end": 3719
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3712,
                                                  "end": 3719
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 3744,
                                                    "end": 3749
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3744,
                                                  "end": 3749
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 3774,
                                                    "end": 3784
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3774,
                                                  "end": 3784
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3524,
                                              "end": 3806
                                            }
                                          },
                                          "loc": {
                                            "start": 3510,
                                            "end": 3806
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3261,
                                        "end": 3824
                                      }
                                    },
                                    "loc": {
                                      "start": 3251,
                                      "end": 3824
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3160,
                                  "end": 3838
                                }
                              },
                              "loc": {
                                "start": 3147,
                                "end": 3838
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 3851,
                                  "end": 3863
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
                                        "start": 3882,
                                        "end": 3884
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3882,
                                      "end": 3884
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3901,
                                        "end": 3911
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3901,
                                      "end": 3911
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 3928,
                                        "end": 3940
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
                                              "start": 3963,
                                              "end": 3965
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3963,
                                            "end": 3965
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 3986,
                                              "end": 3994
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3986,
                                            "end": 3994
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4015,
                                              "end": 4026
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4015,
                                            "end": 4026
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4047,
                                              "end": 4051
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4047,
                                            "end": 4051
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3941,
                                        "end": 4069
                                      }
                                    },
                                    "loc": {
                                      "start": 3928,
                                      "end": 4069
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 4086,
                                        "end": 4095
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
                                              "start": 4118,
                                              "end": 4120
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4118,
                                            "end": 4120
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4141,
                                              "end": 4146
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4141,
                                            "end": 4146
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 4167,
                                              "end": 4171
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4167,
                                            "end": 4171
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 4192,
                                              "end": 4199
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4192,
                                            "end": 4199
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 4220,
                                              "end": 4232
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
                                                    "start": 4259,
                                                    "end": 4261
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4259,
                                                  "end": 4261
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 4286,
                                                    "end": 4294
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4286,
                                                  "end": 4294
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4319,
                                                    "end": 4330
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4319,
                                                  "end": 4330
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4355,
                                                    "end": 4359
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4355,
                                                  "end": 4359
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4233,
                                              "end": 4381
                                            }
                                          },
                                          "loc": {
                                            "start": 4220,
                                            "end": 4381
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4096,
                                        "end": 4399
                                      }
                                    },
                                    "loc": {
                                      "start": 4086,
                                      "end": 4399
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3864,
                                  "end": 4413
                                }
                              },
                              "loc": {
                                "start": 3851,
                                "end": 4413
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 4426,
                                  "end": 4434
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
                                        "start": 4456,
                                        "end": 4471
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 4453,
                                      "end": 4471
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4435,
                                  "end": 4485
                                }
                              },
                              "loc": {
                                "start": 4426,
                                "end": 4485
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4498,
                                  "end": 4500
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4498,
                                "end": 4500
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4513,
                                  "end": 4517
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4513,
                                "end": 4517
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 4530,
                                  "end": 4541
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4530,
                                "end": 4541
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "you",
                                "loc": {
                                  "start": 4554,
                                  "end": 4557
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
                                      "value": "canDelete",
                                      "loc": {
                                        "start": 4576,
                                        "end": 4585
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4576,
                                      "end": 4585
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canRead",
                                      "loc": {
                                        "start": 4602,
                                        "end": 4609
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4602,
                                      "end": 4609
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "canUpdate",
                                      "loc": {
                                        "start": 4626,
                                        "end": 4635
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4626,
                                      "end": 4635
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4558,
                                  "end": 4649
                                }
                              },
                              "loc": {
                                "start": 4554,
                                "end": 4649
                              }
                            }
                          ],
                          "loc": {
                            "start": 374,
                            "end": 4659
                          }
                        },
                        "loc": {
                          "start": 369,
                          "end": 4659
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 4668,
                            "end": 4681
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4668,
                          "end": 4681
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 4690,
                            "end": 4698
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4690,
                          "end": 4698
                        }
                      }
                    ],
                    "loc": {
                      "start": 359,
                      "end": 4704
                    }
                  },
                  "loc": {
                    "start": 343,
                    "end": 4704
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 4709,
                      "end": 4718
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4709,
                    "end": 4718
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 4723,
                      "end": 4736
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
                            "start": 4747,
                            "end": 4749
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4747,
                          "end": 4749
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 4758,
                            "end": 4768
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4758,
                          "end": 4768
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 4777,
                            "end": 4787
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4777,
                          "end": 4787
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 4796,
                            "end": 4801
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4796,
                          "end": 4801
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 4810,
                            "end": 4824
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 4810,
                          "end": 4824
                        }
                      }
                    ],
                    "loc": {
                      "start": 4737,
                      "end": 4830
                    }
                  },
                  "loc": {
                    "start": 4723,
                    "end": 4830
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "codesCount",
                    "loc": {
                      "start": 4835,
                      "end": 4845
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 4835,
                    "end": 4845
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 4850,
                      "end": 4860
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
                            "start": 4871,
                            "end": 4878
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
                                  "start": 4893,
                                  "end": 4895
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4893,
                                "end": 4895
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 4908,
                                  "end": 4918
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4908,
                                "end": 4918
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 4931,
                                  "end": 4934
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
                                        "start": 4953,
                                        "end": 4955
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4953,
                                      "end": 4955
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4972,
                                        "end": 4982
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4972,
                                      "end": 4982
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 4999,
                                        "end": 5002
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4999,
                                      "end": 5002
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 5019,
                                        "end": 5028
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5019,
                                      "end": 5028
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5045,
                                        "end": 5057
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
                                              "start": 5080,
                                              "end": 5082
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5080,
                                            "end": 5082
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5103,
                                              "end": 5111
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5103,
                                            "end": 5111
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5132,
                                              "end": 5143
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5132,
                                            "end": 5143
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5058,
                                        "end": 5161
                                      }
                                    },
                                    "loc": {
                                      "start": 5045,
                                      "end": 5161
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5178,
                                        "end": 5181
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
                                              "start": 5204,
                                              "end": 5209
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5204,
                                            "end": 5209
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5230,
                                              "end": 5242
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5230,
                                            "end": 5242
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5182,
                                        "end": 5260
                                      }
                                    },
                                    "loc": {
                                      "start": 5178,
                                      "end": 5260
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4935,
                                  "end": 5274
                                }
                              },
                              "loc": {
                                "start": 4931,
                                "end": 5274
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 5287,
                                  "end": 5296
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
                                        "start": 5315,
                                        "end": 5321
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
                                              "start": 5344,
                                              "end": 5346
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5344,
                                            "end": 5346
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 5367,
                                              "end": 5372
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5367,
                                            "end": 5372
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 5393,
                                              "end": 5398
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5393,
                                            "end": 5398
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5322,
                                        "end": 5416
                                      }
                                    },
                                    "loc": {
                                      "start": 5315,
                                      "end": 5416
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 5433,
                                        "end": 5445
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
                                              "start": 5468,
                                              "end": 5470
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5468,
                                            "end": 5470
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 5491,
                                              "end": 5501
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5491,
                                            "end": 5501
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 5522,
                                              "end": 5532
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5522,
                                            "end": 5532
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 5553,
                                              "end": 5562
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
                                                    "start": 5589,
                                                    "end": 5591
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5589,
                                                  "end": 5591
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 5616,
                                                    "end": 5626
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5616,
                                                  "end": 5626
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 5651,
                                                    "end": 5661
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5651,
                                                  "end": 5661
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 5686,
                                                    "end": 5690
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5686,
                                                  "end": 5690
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 5715,
                                                    "end": 5726
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5715,
                                                  "end": 5726
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 5751,
                                                    "end": 5758
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5751,
                                                  "end": 5758
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 5783,
                                                    "end": 5788
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5783,
                                                  "end": 5788
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 5813,
                                                    "end": 5823
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 5813,
                                                  "end": 5823
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 5848,
                                                    "end": 5861
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
                                                          "start": 5892,
                                                          "end": 5894
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5892,
                                                        "end": 5894
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 5923,
                                                          "end": 5933
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5923,
                                                        "end": 5933
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 5962,
                                                          "end": 5972
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 5962,
                                                        "end": 5972
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6001,
                                                          "end": 6005
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6001,
                                                        "end": 6005
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6034,
                                                          "end": 6045
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6034,
                                                        "end": 6045
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 6074,
                                                          "end": 6081
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6074,
                                                        "end": 6081
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 6110,
                                                          "end": 6115
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6110,
                                                        "end": 6115
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 6144,
                                                          "end": 6154
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6144,
                                                        "end": 6154
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 5862,
                                                    "end": 6180
                                                  }
                                                },
                                                "loc": {
                                                  "start": 5848,
                                                  "end": 6180
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 5563,
                                              "end": 6202
                                            }
                                          },
                                          "loc": {
                                            "start": 5553,
                                            "end": 6202
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5446,
                                        "end": 6220
                                      }
                                    },
                                    "loc": {
                                      "start": 5433,
                                      "end": 6220
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 6237,
                                        "end": 6249
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
                                              "start": 6272,
                                              "end": 6274
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6272,
                                            "end": 6274
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6295,
                                              "end": 6305
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6295,
                                            "end": 6305
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6326,
                                              "end": 6338
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
                                                    "start": 6365,
                                                    "end": 6367
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6365,
                                                  "end": 6367
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 6392,
                                                    "end": 6400
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6392,
                                                  "end": 6400
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6425,
                                                    "end": 6436
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6425,
                                                  "end": 6436
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6461,
                                                    "end": 6465
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6461,
                                                  "end": 6465
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6339,
                                              "end": 6487
                                            }
                                          },
                                          "loc": {
                                            "start": 6326,
                                            "end": 6487
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 6508,
                                              "end": 6517
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
                                                    "start": 6544,
                                                    "end": 6546
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6544,
                                                  "end": 6546
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6571,
                                                    "end": 6576
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6571,
                                                  "end": 6576
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 6601,
                                                    "end": 6605
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6601,
                                                  "end": 6605
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 6630,
                                                    "end": 6637
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6630,
                                                  "end": 6637
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 6662,
                                                    "end": 6674
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
                                                          "start": 6705,
                                                          "end": 6707
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6705,
                                                        "end": 6707
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 6736,
                                                          "end": 6744
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6736,
                                                        "end": 6744
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6773,
                                                          "end": 6784
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6773,
                                                        "end": 6784
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6813,
                                                          "end": 6817
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6813,
                                                        "end": 6817
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 6675,
                                                    "end": 6843
                                                  }
                                                },
                                                "loc": {
                                                  "start": 6662,
                                                  "end": 6843
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6518,
                                              "end": 6865
                                            }
                                          },
                                          "loc": {
                                            "start": 6508,
                                            "end": 6865
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6250,
                                        "end": 6883
                                      }
                                    },
                                    "loc": {
                                      "start": 6237,
                                      "end": 6883
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 6900,
                                        "end": 6908
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
                                              "start": 6934,
                                              "end": 6949
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 6931,
                                            "end": 6949
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6909,
                                        "end": 6967
                                      }
                                    },
                                    "loc": {
                                      "start": 6900,
                                      "end": 6967
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 6984,
                                        "end": 6986
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 6984,
                                      "end": 6986
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7003,
                                        "end": 7007
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7003,
                                      "end": 7007
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7024,
                                        "end": 7035
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7024,
                                      "end": 7035
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 7052,
                                        "end": 7055
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
                                            "value": "canDelete",
                                            "loc": {
                                              "start": 7078,
                                              "end": 7087
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7078,
                                            "end": 7087
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canRead",
                                            "loc": {
                                              "start": 7108,
                                              "end": 7115
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7108,
                                            "end": 7115
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "canUpdate",
                                            "loc": {
                                              "start": 7136,
                                              "end": 7145
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7136,
                                            "end": 7145
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7056,
                                        "end": 7163
                                      }
                                    },
                                    "loc": {
                                      "start": 7052,
                                      "end": 7163
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5297,
                                  "end": 7177
                                }
                              },
                              "loc": {
                                "start": 5287,
                                "end": 7177
                              }
                            }
                          ],
                          "loc": {
                            "start": 4879,
                            "end": 7187
                          }
                        },
                        "loc": {
                          "start": 4871,
                          "end": 7187
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 7196,
                            "end": 7202
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
                                  "start": 7217,
                                  "end": 7219
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7217,
                                "end": 7219
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 7232,
                                  "end": 7237
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7232,
                                "end": 7237
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 7250,
                                  "end": 7255
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7250,
                                "end": 7255
                              }
                            }
                          ],
                          "loc": {
                            "start": 7203,
                            "end": 7265
                          }
                        },
                        "loc": {
                          "start": 7196,
                          "end": 7265
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 7274,
                            "end": 7286
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
                                  "start": 7301,
                                  "end": 7303
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7301,
                                "end": 7303
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7316,
                                  "end": 7326
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7316,
                                "end": 7326
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7339,
                                  "end": 7349
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7339,
                                "end": 7349
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 7362,
                                  "end": 7371
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
                                        "start": 7390,
                                        "end": 7392
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7390,
                                      "end": 7392
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 7409,
                                        "end": 7419
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7409,
                                      "end": 7419
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 7436,
                                        "end": 7446
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7436,
                                      "end": 7446
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7463,
                                        "end": 7467
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7463,
                                      "end": 7467
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7484,
                                        "end": 7495
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7484,
                                      "end": 7495
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 7512,
                                        "end": 7519
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7512,
                                      "end": 7519
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 7536,
                                        "end": 7541
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7536,
                                      "end": 7541
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 7558,
                                        "end": 7568
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7558,
                                      "end": 7568
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 7585,
                                        "end": 7598
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
                                              "start": 7621,
                                              "end": 7623
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7621,
                                            "end": 7623
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 7644,
                                              "end": 7654
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7644,
                                            "end": 7654
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 7675,
                                              "end": 7685
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7675,
                                            "end": 7685
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 7706,
                                              "end": 7710
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7706,
                                            "end": 7710
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 7731,
                                              "end": 7742
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7731,
                                            "end": 7742
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 7763,
                                              "end": 7770
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7763,
                                            "end": 7770
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 7791,
                                              "end": 7796
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7791,
                                            "end": 7796
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 7817,
                                              "end": 7827
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 7817,
                                            "end": 7827
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7599,
                                        "end": 7845
                                      }
                                    },
                                    "loc": {
                                      "start": 7585,
                                      "end": 7845
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7372,
                                  "end": 7859
                                }
                              },
                              "loc": {
                                "start": 7362,
                                "end": 7859
                              }
                            }
                          ],
                          "loc": {
                            "start": 7287,
                            "end": 7869
                          }
                        },
                        "loc": {
                          "start": 7274,
                          "end": 7869
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 7878,
                            "end": 7890
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
                                  "start": 7905,
                                  "end": 7907
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7905,
                                "end": 7907
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7920,
                                  "end": 7930
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7920,
                                "end": 7930
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 7943,
                                  "end": 7955
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
                                        "start": 7974,
                                        "end": 7976
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7974,
                                      "end": 7976
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 7993,
                                        "end": 8001
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7993,
                                      "end": 8001
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8018,
                                        "end": 8029
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8018,
                                      "end": 8029
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8046,
                                        "end": 8050
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8046,
                                      "end": 8050
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 7956,
                                  "end": 8064
                                }
                              },
                              "loc": {
                                "start": 7943,
                                "end": 8064
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 8077,
                                  "end": 8086
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
                                        "start": 8105,
                                        "end": 8107
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8105,
                                      "end": 8107
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8124,
                                        "end": 8129
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8124,
                                      "end": 8129
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 8146,
                                        "end": 8150
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8146,
                                      "end": 8150
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 8167,
                                        "end": 8174
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8167,
                                      "end": 8174
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 8191,
                                        "end": 8203
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
                                              "start": 8226,
                                              "end": 8228
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8226,
                                            "end": 8228
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 8249,
                                              "end": 8257
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8249,
                                            "end": 8257
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 8278,
                                              "end": 8289
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8278,
                                            "end": 8289
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 8310,
                                              "end": 8314
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8310,
                                            "end": 8314
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8204,
                                        "end": 8332
                                      }
                                    },
                                    "loc": {
                                      "start": 8191,
                                      "end": 8332
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8087,
                                  "end": 8346
                                }
                              },
                              "loc": {
                                "start": 8077,
                                "end": 8346
                              }
                            }
                          ],
                          "loc": {
                            "start": 7891,
                            "end": 8356
                          }
                        },
                        "loc": {
                          "start": 7878,
                          "end": 8356
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 8365,
                            "end": 8373
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
                                  "start": 8391,
                                  "end": 8406
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 8388,
                                "end": 8406
                              }
                            }
                          ],
                          "loc": {
                            "start": 8374,
                            "end": 8416
                          }
                        },
                        "loc": {
                          "start": 8365,
                          "end": 8416
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 8425,
                            "end": 8427
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8425,
                          "end": 8427
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 8436,
                            "end": 8440
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8436,
                          "end": 8440
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 8449,
                            "end": 8460
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 8449,
                          "end": 8460
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "you",
                          "loc": {
                            "start": 8469,
                            "end": 8472
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
                                "value": "canDelete",
                                "loc": {
                                  "start": 8487,
                                  "end": 8496
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8487,
                                "end": 8496
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canRead",
                                "loc": {
                                  "start": 8509,
                                  "end": 8516
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8509,
                                "end": 8516
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "canUpdate",
                                "loc": {
                                  "start": 8529,
                                  "end": 8538
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8529,
                                "end": 8538
                              }
                            }
                          ],
                          "loc": {
                            "start": 8473,
                            "end": 8548
                          }
                        },
                        "loc": {
                          "start": 8469,
                          "end": 8548
                        }
                      }
                    ],
                    "loc": {
                      "start": 4861,
                      "end": 8554
                    }
                  },
                  "loc": {
                    "start": 4850,
                    "end": 8554
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 8559,
                      "end": 8565
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8559,
                    "end": 8565
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 8570,
                      "end": 8580
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8570,
                    "end": 8580
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 8585,
                      "end": 8587
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8585,
                    "end": 8587
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 8592,
                      "end": 8601
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8592,
                    "end": 8601
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 8606,
                      "end": 8622
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8606,
                    "end": 8622
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 8627,
                      "end": 8631
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8627,
                    "end": 8631
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 8636,
                      "end": 8646
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8636,
                    "end": 8646
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 8651,
                      "end": 8664
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8651,
                    "end": 8664
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 8669,
                      "end": 8688
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8669,
                    "end": 8688
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 8693,
                      "end": 8706
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8693,
                    "end": 8706
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 8711,
                      "end": 8725
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8711,
                    "end": 8725
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 8730,
                      "end": 8735
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 8730,
                    "end": 8735
                  }
                }
              ],
              "loc": {
                "start": 337,
                "end": 8737
              }
            },
            "loc": {
              "start": 331,
              "end": 8737
            }
          }
        ],
        "loc": {
          "start": 309,
          "end": 8739
        }
      },
      "loc": {
        "start": 276,
        "end": 8739
      }
    },
    "Wallet_common": {
      "kind": "FragmentDefinition",
      "name": {
        "kind": "Name",
        "value": "Wallet_common",
        "loc": {
          "start": 8749,
          "end": 8762
        }
      },
      "typeCondition": {
        "kind": "NamedType",
        "name": {
          "kind": "Name",
          "value": "Wallet",
          "loc": {
            "start": 8766,
            "end": 8772
          }
        },
        "loc": {
          "start": 8766,
          "end": 8772
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
                "start": 8775,
                "end": 8777
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8775,
              "end": 8777
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "name",
              "loc": {
                "start": 8778,
                "end": 8782
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8778,
              "end": 8782
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "publicAddress",
              "loc": {
                "start": 8783,
                "end": 8796
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8783,
              "end": 8796
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "stakingAddress",
              "loc": {
                "start": 8797,
                "end": 8811
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8797,
              "end": 8811
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "verified",
              "loc": {
                "start": 8812,
                "end": 8820
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 8812,
              "end": 8820
            }
          }
        ],
        "loc": {
          "start": 8773,
          "end": 8822
        }
      },
      "loc": {
        "start": 8740,
        "end": 8822
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
        "start": 8833,
        "end": 8847
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
              "start": 8849,
              "end": 8854
            }
          },
          "loc": {
            "start": 8848,
            "end": 8854
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
                "start": 8856,
                "end": 8875
              }
            },
            "loc": {
              "start": 8856,
              "end": 8875
            }
          },
          "loc": {
            "start": 8856,
            "end": 8876
          }
        },
        "directives": [],
        "loc": {
          "start": 8848,
          "end": 8876
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
              "start": 8882,
              "end": 8896
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 8897,
                  "end": 8902
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 8905,
                    "end": 8910
                  }
                },
                "loc": {
                  "start": 8904,
                  "end": 8910
                }
              },
              "loc": {
                "start": 8897,
                "end": 8910
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
                    "start": 8918,
                    "end": 8928
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 8918,
                  "end": 8928
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "session",
                  "loc": {
                    "start": 8933,
                    "end": 8940
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
                          "start": 8954,
                          "end": 8966
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8951,
                        "end": 8966
                      }
                    }
                  ],
                  "loc": {
                    "start": 8941,
                    "end": 8972
                  }
                },
                "loc": {
                  "start": 8933,
                  "end": 8972
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "wallet",
                  "loc": {
                    "start": 8977,
                    "end": 8983
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
                          "start": 8997,
                          "end": 9010
                        }
                      },
                      "directives": [],
                      "loc": {
                        "start": 8994,
                        "end": 9010
                      }
                    }
                  ],
                  "loc": {
                    "start": 8984,
                    "end": 9016
                  }
                },
                "loc": {
                  "start": 8977,
                  "end": 9016
                }
              }
            ],
            "loc": {
              "start": 8912,
              "end": 9020
            }
          },
          "loc": {
            "start": 8882,
            "end": 9020
          }
        }
      ],
      "loc": {
        "start": 8878,
        "end": 9022
      }
    },
    "loc": {
      "start": 8824,
      "end": 9022
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_walletComplete"
  }
} as const;
