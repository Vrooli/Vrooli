export const auth_emailResetPassword = {
  "fieldName": "emailResetPassword",
  "fieldNodes": [
    {
      "kind": "Field",
      "name": {
        "kind": "Name",
        "value": "emailResetPassword",
        "loc": {
          "start": 343,
          "end": 361
        }
      },
      "arguments": [
        {
          "kind": "Argument",
          "name": {
            "kind": "Name",
            "value": "input",
            "loc": {
              "start": 362,
              "end": 367
            }
          },
          "value": {
            "kind": "Variable",
            "name": {
              "kind": "Name",
              "value": "input",
              "loc": {
                "start": 370,
                "end": 375
              }
            },
            "loc": {
              "start": 369,
              "end": 375
            }
          },
          "loc": {
            "start": 362,
            "end": 375
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
              "value": "isLoggedIn",
              "loc": {
                "start": 383,
                "end": 393
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 383,
              "end": 393
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "timeZone",
              "loc": {
                "start": 398,
                "end": 406
              }
            },
            "arguments": [],
            "directives": [],
            "loc": {
              "start": 398,
              "end": 406
            }
          },
          {
            "kind": "Field",
            "name": {
              "kind": "Name",
              "value": "users",
              "loc": {
                "start": 411,
                "end": 416
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
                      "start": 427,
                      "end": 442
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
                            "start": 457,
                            "end": 461
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
                                  "start": 480,
                                  "end": 487
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
                                        "start": 510,
                                        "end": 512
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 510,
                                      "end": 512
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "filterType",
                                      "loc": {
                                        "start": 533,
                                        "end": 543
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 533,
                                      "end": 543
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 564,
                                        "end": 567
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
                                              "start": 594,
                                              "end": 596
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 594,
                                            "end": 596
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 621,
                                              "end": 631
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 621,
                                            "end": 631
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "tag",
                                            "loc": {
                                              "start": 656,
                                              "end": 659
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 656,
                                            "end": 659
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "bookmarks",
                                            "loc": {
                                              "start": 684,
                                              "end": 693
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 684,
                                            "end": 693
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 718,
                                              "end": 730
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
                                                    "start": 761,
                                                    "end": 763
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 761,
                                                  "end": 763
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 792,
                                                    "end": 800
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 792,
                                                  "end": 800
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 829,
                                                    "end": 840
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 829,
                                                  "end": 840
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 731,
                                              "end": 866
                                            }
                                          },
                                          "loc": {
                                            "start": 718,
                                            "end": 866
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "you",
                                            "loc": {
                                              "start": 891,
                                              "end": 894
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
                                                    "start": 925,
                                                    "end": 930
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 925,
                                                  "end": 930
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isBookmarked",
                                                  "loc": {
                                                    "start": 959,
                                                    "end": 971
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 959,
                                                  "end": 971
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 895,
                                              "end": 997
                                            }
                                          },
                                          "loc": {
                                            "start": 891,
                                            "end": 997
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 568,
                                        "end": 1019
                                      }
                                    },
                                    "loc": {
                                      "start": 564,
                                      "end": 1019
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "focusMode",
                                      "loc": {
                                        "start": 1040,
                                        "end": 1049
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
                                              "start": 1076,
                                              "end": 1082
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
                                                    "start": 1113,
                                                    "end": 1115
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1113,
                                                  "end": 1115
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "color",
                                                  "loc": {
                                                    "start": 1144,
                                                    "end": 1149
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1144,
                                                  "end": 1149
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "label",
                                                  "loc": {
                                                    "start": 1178,
                                                    "end": 1183
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1178,
                                                  "end": 1183
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1083,
                                              "end": 1209
                                            }
                                          },
                                          "loc": {
                                            "start": 1076,
                                            "end": 1209
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderList",
                                            "loc": {
                                              "start": 1234,
                                              "end": 1246
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
                                                    "start": 1277,
                                                    "end": 1279
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1277,
                                                  "end": 1279
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 1308,
                                                    "end": 1318
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1308,
                                                  "end": 1318
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 1347,
                                                    "end": 1357
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 1347,
                                                  "end": 1357
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminders",
                                                  "loc": {
                                                    "start": 1386,
                                                    "end": 1395
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
                                                          "start": 1430,
                                                          "end": 1432
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1430,
                                                        "end": 1432
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 1465,
                                                          "end": 1475
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1465,
                                                        "end": 1475
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 1508,
                                                          "end": 1518
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1508,
                                                        "end": 1518
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 1551,
                                                          "end": 1555
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1551,
                                                        "end": 1555
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 1588,
                                                          "end": 1599
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1588,
                                                        "end": 1599
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 1632,
                                                          "end": 1639
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1632,
                                                        "end": 1639
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 1672,
                                                          "end": 1677
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1672,
                                                        "end": 1677
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 1710,
                                                          "end": 1720
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 1710,
                                                        "end": 1720
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "reminderItems",
                                                        "loc": {
                                                          "start": 1753,
                                                          "end": 1766
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
                                                                "start": 1805,
                                                                "end": 1807
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1805,
                                                              "end": 1807
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "created_at",
                                                              "loc": {
                                                                "start": 1844,
                                                                "end": 1854
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1844,
                                                              "end": 1854
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "updated_at",
                                                              "loc": {
                                                                "start": 1891,
                                                                "end": 1901
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1891,
                                                              "end": 1901
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 1938,
                                                                "end": 1942
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1938,
                                                              "end": 1942
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 1979,
                                                                "end": 1990
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 1979,
                                                              "end": 1990
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "dueDate",
                                                              "loc": {
                                                                "start": 2027,
                                                                "end": 2034
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2027,
                                                              "end": 2034
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "index",
                                                              "loc": {
                                                                "start": 2071,
                                                                "end": 2076
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2071,
                                                              "end": 2076
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "isComplete",
                                                              "loc": {
                                                                "start": 2113,
                                                                "end": 2123
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2113,
                                                              "end": 2123
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 1767,
                                                          "end": 2157
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 1753,
                                                        "end": 2157
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 1396,
                                                    "end": 2187
                                                  }
                                                },
                                                "loc": {
                                                  "start": 1386,
                                                  "end": 2187
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 1247,
                                              "end": 2213
                                            }
                                          },
                                          "loc": {
                                            "start": 1234,
                                            "end": 2213
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resourceList",
                                            "loc": {
                                              "start": 2238,
                                              "end": 2250
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
                                                    "start": 2281,
                                                    "end": 2283
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2281,
                                                  "end": 2283
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 2312,
                                                    "end": 2322
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 2312,
                                                  "end": 2322
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 2351,
                                                    "end": 2363
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
                                                          "start": 2398,
                                                          "end": 2400
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2398,
                                                        "end": 2400
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 2433,
                                                          "end": 2441
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2433,
                                                        "end": 2441
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 2474,
                                                          "end": 2485
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2474,
                                                        "end": 2485
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 2518,
                                                          "end": 2522
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2518,
                                                        "end": 2522
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2364,
                                                    "end": 2552
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2351,
                                                  "end": 2552
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "resources",
                                                  "loc": {
                                                    "start": 2581,
                                                    "end": 2590
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
                                                          "start": 2625,
                                                          "end": 2627
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2625,
                                                        "end": 2627
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 2660,
                                                          "end": 2665
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2660,
                                                        "end": 2665
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "link",
                                                        "loc": {
                                                          "start": 2698,
                                                          "end": 2702
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2698,
                                                        "end": 2702
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "usedFor",
                                                        "loc": {
                                                          "start": 2735,
                                                          "end": 2742
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 2735,
                                                        "end": 2742
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "translations",
                                                        "loc": {
                                                          "start": 2775,
                                                          "end": 2787
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
                                                                "start": 2826,
                                                                "end": 2828
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2826,
                                                              "end": 2828
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "language",
                                                              "loc": {
                                                                "start": 2865,
                                                                "end": 2873
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2865,
                                                              "end": 2873
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "description",
                                                              "loc": {
                                                                "start": 2910,
                                                                "end": 2921
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2910,
                                                              "end": 2921
                                                            }
                                                          },
                                                          {
                                                            "kind": "Field",
                                                            "name": {
                                                              "kind": "Name",
                                                              "value": "name",
                                                              "loc": {
                                                                "start": 2958,
                                                                "end": 2962
                                                              }
                                                            },
                                                            "arguments": [],
                                                            "directives": [],
                                                            "loc": {
                                                              "start": 2958,
                                                              "end": 2962
                                                            }
                                                          }
                                                        ],
                                                        "loc": {
                                                          "start": 2788,
                                                          "end": 2996
                                                        }
                                                      },
                                                      "loc": {
                                                        "start": 2775,
                                                        "end": 2996
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 2591,
                                                    "end": 3026
                                                  }
                                                },
                                                "loc": {
                                                  "start": 2581,
                                                  "end": 3026
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 2251,
                                              "end": 3052
                                            }
                                          },
                                          "loc": {
                                            "start": 2238,
                                            "end": 3052
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "schedule",
                                            "loc": {
                                              "start": 3077,
                                              "end": 3085
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
                                                    "start": 3119,
                                                    "end": 3134
                                                  }
                                                },
                                                "directives": [],
                                                "loc": {
                                                  "start": 3116,
                                                  "end": 3134
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3086,
                                              "end": 3160
                                            }
                                          },
                                          "loc": {
                                            "start": 3077,
                                            "end": 3160
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "id",
                                            "loc": {
                                              "start": 3185,
                                              "end": 3187
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3185,
                                            "end": 3187
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3212,
                                              "end": 3216
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3212,
                                            "end": 3216
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3241,
                                              "end": 3252
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3241,
                                            "end": 3252
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 1050,
                                        "end": 3274
                                      }
                                    },
                                    "loc": {
                                      "start": 1040,
                                      "end": 3274
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 488,
                                  "end": 3292
                                }
                              },
                              "loc": {
                                "start": 480,
                                "end": 3292
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "labels",
                                "loc": {
                                  "start": 3309,
                                  "end": 3315
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
                                        "start": 3338,
                                        "end": 3340
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3338,
                                      "end": 3340
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "color",
                                      "loc": {
                                        "start": 3361,
                                        "end": 3366
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3361,
                                      "end": 3366
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "label",
                                      "loc": {
                                        "start": 3387,
                                        "end": 3392
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3387,
                                      "end": 3392
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3316,
                                  "end": 3410
                                }
                              },
                              "loc": {
                                "start": 3309,
                                "end": 3410
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminderList",
                                "loc": {
                                  "start": 3427,
                                  "end": 3439
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
                                        "start": 3462,
                                        "end": 3464
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3462,
                                      "end": 3464
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 3485,
                                        "end": 3495
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3485,
                                      "end": 3495
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 3516,
                                        "end": 3526
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 3516,
                                      "end": 3526
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminders",
                                      "loc": {
                                        "start": 3547,
                                        "end": 3556
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
                                              "start": 3583,
                                              "end": 3585
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3583,
                                            "end": 3585
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 3610,
                                              "end": 3620
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3610,
                                            "end": 3620
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 3645,
                                              "end": 3655
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3645,
                                            "end": 3655
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 3680,
                                              "end": 3684
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3680,
                                            "end": 3684
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 3709,
                                              "end": 3720
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3709,
                                            "end": 3720
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 3745,
                                              "end": 3752
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3745,
                                            "end": 3752
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 3777,
                                              "end": 3782
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3777,
                                            "end": 3782
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 3807,
                                              "end": 3817
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 3807,
                                            "end": 3817
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminderItems",
                                            "loc": {
                                              "start": 3842,
                                              "end": 3855
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
                                                    "start": 3886,
                                                    "end": 3888
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3886,
                                                  "end": 3888
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 3917,
                                                    "end": 3927
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3917,
                                                  "end": 3927
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 3956,
                                                    "end": 3966
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3956,
                                                  "end": 3966
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 3995,
                                                    "end": 3999
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 3995,
                                                  "end": 3999
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4028,
                                                    "end": 4039
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4028,
                                                  "end": 4039
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 4068,
                                                    "end": 4075
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4068,
                                                  "end": 4075
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 4104,
                                                    "end": 4109
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4104,
                                                  "end": 4109
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 4138,
                                                    "end": 4148
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4138,
                                                  "end": 4148
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 3856,
                                              "end": 4174
                                            }
                                          },
                                          "loc": {
                                            "start": 3842,
                                            "end": 4174
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 3557,
                                        "end": 4196
                                      }
                                    },
                                    "loc": {
                                      "start": 3547,
                                      "end": 4196
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 3440,
                                  "end": 4214
                                }
                              },
                              "loc": {
                                "start": 3427,
                                "end": 4214
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resourceList",
                                "loc": {
                                  "start": 4231,
                                  "end": 4243
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
                                        "start": 4266,
                                        "end": 4268
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4266,
                                      "end": 4268
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 4289,
                                        "end": 4299
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 4289,
                                      "end": 4299
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 4320,
                                        "end": 4332
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
                                              "start": 4359,
                                              "end": 4361
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4359,
                                            "end": 4361
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 4386,
                                              "end": 4394
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4386,
                                            "end": 4394
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 4419,
                                              "end": 4430
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4419,
                                            "end": 4430
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 4455,
                                              "end": 4459
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4455,
                                            "end": 4459
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4333,
                                        "end": 4481
                                      }
                                    },
                                    "loc": {
                                      "start": 4320,
                                      "end": 4481
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resources",
                                      "loc": {
                                        "start": 4502,
                                        "end": 4511
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
                                              "start": 4538,
                                              "end": 4540
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4538,
                                            "end": 4540
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 4565,
                                              "end": 4570
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4565,
                                            "end": 4570
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "link",
                                            "loc": {
                                              "start": 4595,
                                              "end": 4599
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4595,
                                            "end": 4599
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "usedFor",
                                            "loc": {
                                              "start": 4624,
                                              "end": 4631
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 4624,
                                            "end": 4631
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 4656,
                                              "end": 4668
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
                                                    "start": 4699,
                                                    "end": 4701
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4699,
                                                  "end": 4701
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 4730,
                                                    "end": 4738
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4730,
                                                  "end": 4738
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 4767,
                                                    "end": 4778
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4767,
                                                  "end": 4778
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 4807,
                                                    "end": 4811
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 4807,
                                                  "end": 4811
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 4669,
                                              "end": 4837
                                            }
                                          },
                                          "loc": {
                                            "start": 4656,
                                            "end": 4837
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 4512,
                                        "end": 4859
                                      }
                                    },
                                    "loc": {
                                      "start": 4502,
                                      "end": 4859
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4244,
                                  "end": 4877
                                }
                              },
                              "loc": {
                                "start": 4231,
                                "end": 4877
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "schedule",
                                "loc": {
                                  "start": 4894,
                                  "end": 4902
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
                                        "start": 4928,
                                        "end": 4943
                                      }
                                    },
                                    "directives": [],
                                    "loc": {
                                      "start": 4925,
                                      "end": 4943
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 4903,
                                  "end": 4961
                                }
                              },
                              "loc": {
                                "start": 4894,
                                "end": 4961
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "id",
                                "loc": {
                                  "start": 4978,
                                  "end": 4980
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4978,
                                "end": 4980
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "name",
                                "loc": {
                                  "start": 4997,
                                  "end": 5001
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 4997,
                                "end": 5001
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "description",
                                "loc": {
                                  "start": 5018,
                                  "end": 5029
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5018,
                                "end": 5029
                              }
                            }
                          ],
                          "loc": {
                            "start": 462,
                            "end": 5043
                          }
                        },
                        "loc": {
                          "start": 457,
                          "end": 5043
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopCondition",
                          "loc": {
                            "start": 5056,
                            "end": 5069
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5056,
                          "end": 5069
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "stopTime",
                          "loc": {
                            "start": 5082,
                            "end": 5090
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5082,
                          "end": 5090
                        }
                      }
                    ],
                    "loc": {
                      "start": 443,
                      "end": 5100
                    }
                  },
                  "loc": {
                    "start": 427,
                    "end": 5100
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "apisCount",
                    "loc": {
                      "start": 5109,
                      "end": 5118
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 5109,
                    "end": 5118
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "bookmarkLists",
                    "loc": {
                      "start": 5127,
                      "end": 5140
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
                            "start": 5155,
                            "end": 5157
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5155,
                          "end": 5157
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "created_at",
                          "loc": {
                            "start": 5170,
                            "end": 5180
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5170,
                          "end": 5180
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "updated_at",
                          "loc": {
                            "start": 5193,
                            "end": 5203
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5193,
                          "end": 5203
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "label",
                          "loc": {
                            "start": 5216,
                            "end": 5221
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5216,
                          "end": 5221
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "bookmarksCount",
                          "loc": {
                            "start": 5234,
                            "end": 5248
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 5234,
                          "end": 5248
                        }
                      }
                    ],
                    "loc": {
                      "start": 5141,
                      "end": 5258
                    }
                  },
                  "loc": {
                    "start": 5127,
                    "end": 5258
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "focusModes",
                    "loc": {
                      "start": 5267,
                      "end": 5277
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
                            "start": 5292,
                            "end": 5299
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
                                  "start": 5318,
                                  "end": 5320
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5318,
                                "end": 5320
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "filterType",
                                "loc": {
                                  "start": 5337,
                                  "end": 5347
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 5337,
                                "end": 5347
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "tag",
                                "loc": {
                                  "start": 5364,
                                  "end": 5367
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
                                        "start": 5390,
                                        "end": 5392
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5390,
                                      "end": 5392
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 5413,
                                        "end": 5423
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5413,
                                      "end": 5423
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "tag",
                                      "loc": {
                                        "start": 5444,
                                        "end": 5447
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5444,
                                      "end": 5447
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "bookmarks",
                                      "loc": {
                                        "start": 5468,
                                        "end": 5477
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 5468,
                                      "end": 5477
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 5498,
                                        "end": 5510
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
                                              "start": 5537,
                                              "end": 5539
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5537,
                                            "end": 5539
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 5564,
                                              "end": 5572
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5564,
                                            "end": 5572
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 5597,
                                              "end": 5608
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5597,
                                            "end": 5608
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5511,
                                        "end": 5630
                                      }
                                    },
                                    "loc": {
                                      "start": 5498,
                                      "end": 5630
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "you",
                                      "loc": {
                                        "start": 5651,
                                        "end": 5654
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
                                              "start": 5681,
                                              "end": 5686
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5681,
                                            "end": 5686
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isBookmarked",
                                            "loc": {
                                              "start": 5711,
                                              "end": 5723
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5711,
                                            "end": 5723
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5655,
                                        "end": 5745
                                      }
                                    },
                                    "loc": {
                                      "start": 5651,
                                      "end": 5745
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5368,
                                  "end": 5763
                                }
                              },
                              "loc": {
                                "start": 5364,
                                "end": 5763
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "focusMode",
                                "loc": {
                                  "start": 5780,
                                  "end": 5789
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
                                        "start": 5812,
                                        "end": 5818
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
                                              "start": 5845,
                                              "end": 5847
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5845,
                                            "end": 5847
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "color",
                                            "loc": {
                                              "start": 5872,
                                              "end": 5877
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5872,
                                            "end": 5877
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "label",
                                            "loc": {
                                              "start": 5902,
                                              "end": 5907
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5902,
                                            "end": 5907
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5819,
                                        "end": 5929
                                      }
                                    },
                                    "loc": {
                                      "start": 5812,
                                      "end": 5929
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderList",
                                      "loc": {
                                        "start": 5950,
                                        "end": 5962
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
                                              "start": 5989,
                                              "end": 5991
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 5989,
                                            "end": 5991
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6016,
                                              "end": 6026
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6016,
                                            "end": 6026
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 6051,
                                              "end": 6061
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6051,
                                            "end": 6061
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "reminders",
                                            "loc": {
                                              "start": 6086,
                                              "end": 6095
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
                                                    "start": 6126,
                                                    "end": 6128
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6126,
                                                  "end": 6128
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "created_at",
                                                  "loc": {
                                                    "start": 6157,
                                                    "end": 6167
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6157,
                                                  "end": 6167
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "updated_at",
                                                  "loc": {
                                                    "start": 6196,
                                                    "end": 6206
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6196,
                                                  "end": 6206
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 6235,
                                                    "end": 6239
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6235,
                                                  "end": 6239
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 6268,
                                                    "end": 6279
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6268,
                                                  "end": 6279
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "dueDate",
                                                  "loc": {
                                                    "start": 6308,
                                                    "end": 6315
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6308,
                                                  "end": 6315
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
                                                  "loc": {
                                                    "start": 6344,
                                                    "end": 6349
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6344,
                                                  "end": 6349
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "isComplete",
                                                  "loc": {
                                                    "start": 6378,
                                                    "end": 6388
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6378,
                                                  "end": 6388
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "reminderItems",
                                                  "loc": {
                                                    "start": 6417,
                                                    "end": 6430
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
                                                          "start": 6465,
                                                          "end": 6467
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6465,
                                                        "end": 6467
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "created_at",
                                                        "loc": {
                                                          "start": 6500,
                                                          "end": 6510
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6500,
                                                        "end": 6510
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "updated_at",
                                                        "loc": {
                                                          "start": 6543,
                                                          "end": 6553
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6543,
                                                        "end": 6553
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 6586,
                                                          "end": 6590
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6586,
                                                        "end": 6590
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 6623,
                                                          "end": 6634
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6623,
                                                        "end": 6634
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "dueDate",
                                                        "loc": {
                                                          "start": 6667,
                                                          "end": 6674
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6667,
                                                        "end": 6674
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "index",
                                                        "loc": {
                                                          "start": 6707,
                                                          "end": 6712
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6707,
                                                        "end": 6712
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "isComplete",
                                                        "loc": {
                                                          "start": 6745,
                                                          "end": 6755
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 6745,
                                                        "end": 6755
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 6431,
                                                    "end": 6785
                                                  }
                                                },
                                                "loc": {
                                                  "start": 6417,
                                                  "end": 6785
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6096,
                                              "end": 6811
                                            }
                                          },
                                          "loc": {
                                            "start": 6086,
                                            "end": 6811
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 5963,
                                        "end": 6833
                                      }
                                    },
                                    "loc": {
                                      "start": 5950,
                                      "end": 6833
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "resourceList",
                                      "loc": {
                                        "start": 6854,
                                        "end": 6866
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
                                              "start": 6893,
                                              "end": 6895
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6893,
                                            "end": 6895
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 6920,
                                              "end": 6930
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 6920,
                                            "end": 6930
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "translations",
                                            "loc": {
                                              "start": 6955,
                                              "end": 6967
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
                                                    "start": 6998,
                                                    "end": 7000
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 6998,
                                                  "end": 7000
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "language",
                                                  "loc": {
                                                    "start": 7029,
                                                    "end": 7037
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7029,
                                                  "end": 7037
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "description",
                                                  "loc": {
                                                    "start": 7066,
                                                    "end": 7077
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7066,
                                                  "end": 7077
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "name",
                                                  "loc": {
                                                    "start": 7106,
                                                    "end": 7110
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7106,
                                                  "end": 7110
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 6968,
                                              "end": 7136
                                            }
                                          },
                                          "loc": {
                                            "start": 6955,
                                            "end": 7136
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "resources",
                                            "loc": {
                                              "start": 7161,
                                              "end": 7170
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
                                                    "start": 7201,
                                                    "end": 7203
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7201,
                                                  "end": 7203
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "index",
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
                                                  "value": "link",
                                                  "loc": {
                                                    "start": 7266,
                                                    "end": 7270
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7266,
                                                  "end": 7270
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "usedFor",
                                                  "loc": {
                                                    "start": 7299,
                                                    "end": 7306
                                                  }
                                                },
                                                "arguments": [],
                                                "directives": [],
                                                "loc": {
                                                  "start": 7299,
                                                  "end": 7306
                                                }
                                              },
                                              {
                                                "kind": "Field",
                                                "name": {
                                                  "kind": "Name",
                                                  "value": "translations",
                                                  "loc": {
                                                    "start": 7335,
                                                    "end": 7347
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
                                                          "start": 7382,
                                                          "end": 7384
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7382,
                                                        "end": 7384
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "language",
                                                        "loc": {
                                                          "start": 7417,
                                                          "end": 7425
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7417,
                                                        "end": 7425
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "description",
                                                        "loc": {
                                                          "start": 7458,
                                                          "end": 7469
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7458,
                                                        "end": 7469
                                                      }
                                                    },
                                                    {
                                                      "kind": "Field",
                                                      "name": {
                                                        "kind": "Name",
                                                        "value": "name",
                                                        "loc": {
                                                          "start": 7502,
                                                          "end": 7506
                                                        }
                                                      },
                                                      "arguments": [],
                                                      "directives": [],
                                                      "loc": {
                                                        "start": 7502,
                                                        "end": 7506
                                                      }
                                                    }
                                                  ],
                                                  "loc": {
                                                    "start": 7348,
                                                    "end": 7536
                                                  }
                                                },
                                                "loc": {
                                                  "start": 7335,
                                                  "end": 7536
                                                }
                                              }
                                            ],
                                            "loc": {
                                              "start": 7171,
                                              "end": 7562
                                            }
                                          },
                                          "loc": {
                                            "start": 7161,
                                            "end": 7562
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 6867,
                                        "end": 7584
                                      }
                                    },
                                    "loc": {
                                      "start": 6854,
                                      "end": 7584
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "schedule",
                                      "loc": {
                                        "start": 7605,
                                        "end": 7613
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
                                              "start": 7643,
                                              "end": 7658
                                            }
                                          },
                                          "directives": [],
                                          "loc": {
                                            "start": 7640,
                                            "end": 7658
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 7614,
                                        "end": 7680
                                      }
                                    },
                                    "loc": {
                                      "start": 7605,
                                      "end": 7680
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "id",
                                      "loc": {
                                        "start": 7701,
                                        "end": 7703
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7701,
                                      "end": 7703
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 7724,
                                        "end": 7728
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7724,
                                      "end": 7728
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 7749,
                                        "end": 7760
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 7749,
                                      "end": 7760
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 5790,
                                  "end": 7778
                                }
                              },
                              "loc": {
                                "start": 5780,
                                "end": 7778
                              }
                            }
                          ],
                          "loc": {
                            "start": 5300,
                            "end": 7792
                          }
                        },
                        "loc": {
                          "start": 5292,
                          "end": 7792
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "labels",
                          "loc": {
                            "start": 7805,
                            "end": 7811
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
                                  "start": 7830,
                                  "end": 7832
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7830,
                                "end": 7832
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "color",
                                "loc": {
                                  "start": 7849,
                                  "end": 7854
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7849,
                                "end": 7854
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "label",
                                "loc": {
                                  "start": 7871,
                                  "end": 7876
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7871,
                                "end": 7876
                              }
                            }
                          ],
                          "loc": {
                            "start": 7812,
                            "end": 7890
                          }
                        },
                        "loc": {
                          "start": 7805,
                          "end": 7890
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "reminderList",
                          "loc": {
                            "start": 7903,
                            "end": 7915
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
                                  "start": 7934,
                                  "end": 7936
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7934,
                                "end": 7936
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 7953,
                                  "end": 7963
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7953,
                                "end": 7963
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "updated_at",
                                "loc": {
                                  "start": 7980,
                                  "end": 7990
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 7980,
                                "end": 7990
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "reminders",
                                "loc": {
                                  "start": 8007,
                                  "end": 8016
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
                                        "start": 8039,
                                        "end": 8041
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8039,
                                      "end": 8041
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "created_at",
                                      "loc": {
                                        "start": 8062,
                                        "end": 8072
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8062,
                                      "end": 8072
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "updated_at",
                                      "loc": {
                                        "start": 8093,
                                        "end": 8103
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8093,
                                      "end": 8103
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8124,
                                        "end": 8128
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8124,
                                      "end": 8128
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8149,
                                        "end": 8160
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8149,
                                      "end": 8160
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "dueDate",
                                      "loc": {
                                        "start": 8181,
                                        "end": 8188
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8181,
                                      "end": 8188
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8209,
                                        "end": 8214
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8209,
                                      "end": 8214
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "isComplete",
                                      "loc": {
                                        "start": 8235,
                                        "end": 8245
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8235,
                                      "end": 8245
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "reminderItems",
                                      "loc": {
                                        "start": 8266,
                                        "end": 8279
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
                                              "start": 8306,
                                              "end": 8308
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8306,
                                            "end": 8308
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "created_at",
                                            "loc": {
                                              "start": 8333,
                                              "end": 8343
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8333,
                                            "end": 8343
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "updated_at",
                                            "loc": {
                                              "start": 8368,
                                              "end": 8378
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8368,
                                            "end": 8378
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 8403,
                                              "end": 8407
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8403,
                                            "end": 8407
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 8432,
                                              "end": 8443
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8432,
                                            "end": 8443
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "dueDate",
                                            "loc": {
                                              "start": 8468,
                                              "end": 8475
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8468,
                                            "end": 8475
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "index",
                                            "loc": {
                                              "start": 8500,
                                              "end": 8505
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8500,
                                            "end": 8505
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "isComplete",
                                            "loc": {
                                              "start": 8530,
                                              "end": 8540
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 8530,
                                            "end": 8540
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8280,
                                        "end": 8562
                                      }
                                    },
                                    "loc": {
                                      "start": 8266,
                                      "end": 8562
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8017,
                                  "end": 8580
                                }
                              },
                              "loc": {
                                "start": 8007,
                                "end": 8580
                              }
                            }
                          ],
                          "loc": {
                            "start": 7916,
                            "end": 8594
                          }
                        },
                        "loc": {
                          "start": 7903,
                          "end": 8594
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "resourceList",
                          "loc": {
                            "start": 8607,
                            "end": 8619
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
                                  "start": 8638,
                                  "end": 8640
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8638,
                                "end": 8640
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "created_at",
                                "loc": {
                                  "start": 8657,
                                  "end": 8667
                                }
                              },
                              "arguments": [],
                              "directives": [],
                              "loc": {
                                "start": 8657,
                                "end": 8667
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "translations",
                                "loc": {
                                  "start": 8684,
                                  "end": 8696
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
                                        "start": 8719,
                                        "end": 8721
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8719,
                                      "end": 8721
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "language",
                                      "loc": {
                                        "start": 8742,
                                        "end": 8750
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8742,
                                      "end": 8750
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "description",
                                      "loc": {
                                        "start": 8771,
                                        "end": 8782
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8771,
                                      "end": 8782
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "name",
                                      "loc": {
                                        "start": 8803,
                                        "end": 8807
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8803,
                                      "end": 8807
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8697,
                                  "end": 8825
                                }
                              },
                              "loc": {
                                "start": 8684,
                                "end": 8825
                              }
                            },
                            {
                              "kind": "Field",
                              "name": {
                                "kind": "Name",
                                "value": "resources",
                                "loc": {
                                  "start": 8842,
                                  "end": 8851
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
                                        "start": 8874,
                                        "end": 8876
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8874,
                                      "end": 8876
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "index",
                                      "loc": {
                                        "start": 8897,
                                        "end": 8902
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8897,
                                      "end": 8902
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "link",
                                      "loc": {
                                        "start": 8923,
                                        "end": 8927
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8923,
                                      "end": 8927
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "usedFor",
                                      "loc": {
                                        "start": 8948,
                                        "end": 8955
                                      }
                                    },
                                    "arguments": [],
                                    "directives": [],
                                    "loc": {
                                      "start": 8948,
                                      "end": 8955
                                    }
                                  },
                                  {
                                    "kind": "Field",
                                    "name": {
                                      "kind": "Name",
                                      "value": "translations",
                                      "loc": {
                                        "start": 8976,
                                        "end": 8988
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
                                              "start": 9015,
                                              "end": 9017
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9015,
                                            "end": 9017
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "language",
                                            "loc": {
                                              "start": 9042,
                                              "end": 9050
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9042,
                                            "end": 9050
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "description",
                                            "loc": {
                                              "start": 9075,
                                              "end": 9086
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9075,
                                            "end": 9086
                                          }
                                        },
                                        {
                                          "kind": "Field",
                                          "name": {
                                            "kind": "Name",
                                            "value": "name",
                                            "loc": {
                                              "start": 9111,
                                              "end": 9115
                                            }
                                          },
                                          "arguments": [],
                                          "directives": [],
                                          "loc": {
                                            "start": 9111,
                                            "end": 9115
                                          }
                                        }
                                      ],
                                      "loc": {
                                        "start": 8989,
                                        "end": 9137
                                      }
                                    },
                                    "loc": {
                                      "start": 8976,
                                      "end": 9137
                                    }
                                  }
                                ],
                                "loc": {
                                  "start": 8852,
                                  "end": 9155
                                }
                              },
                              "loc": {
                                "start": 8842,
                                "end": 9155
                              }
                            }
                          ],
                          "loc": {
                            "start": 8620,
                            "end": 9169
                          }
                        },
                        "loc": {
                          "start": 8607,
                          "end": 9169
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "schedule",
                          "loc": {
                            "start": 9182,
                            "end": 9190
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
                                  "start": 9212,
                                  "end": 9227
                                }
                              },
                              "directives": [],
                              "loc": {
                                "start": 9209,
                                "end": 9227
                              }
                            }
                          ],
                          "loc": {
                            "start": 9191,
                            "end": 9241
                          }
                        },
                        "loc": {
                          "start": 9182,
                          "end": 9241
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "id",
                          "loc": {
                            "start": 9254,
                            "end": 9256
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9254,
                          "end": 9256
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "name",
                          "loc": {
                            "start": 9269,
                            "end": 9273
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9269,
                          "end": 9273
                        }
                      },
                      {
                        "kind": "Field",
                        "name": {
                          "kind": "Name",
                          "value": "description",
                          "loc": {
                            "start": 9286,
                            "end": 9297
                          }
                        },
                        "arguments": [],
                        "directives": [],
                        "loc": {
                          "start": 9286,
                          "end": 9297
                        }
                      }
                    ],
                    "loc": {
                      "start": 5278,
                      "end": 9307
                    }
                  },
                  "loc": {
                    "start": 5267,
                    "end": 9307
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "handle",
                    "loc": {
                      "start": 9316,
                      "end": 9322
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9316,
                    "end": 9322
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "hasPremium",
                    "loc": {
                      "start": 9331,
                      "end": 9341
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9331,
                    "end": 9341
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "id",
                    "loc": {
                      "start": 9350,
                      "end": 9352
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9350,
                    "end": 9352
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "languages",
                    "loc": {
                      "start": 9361,
                      "end": 9370
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9361,
                    "end": 9370
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "membershipsCount",
                    "loc": {
                      "start": 9379,
                      "end": 9395
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9379,
                    "end": 9395
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "name",
                    "loc": {
                      "start": 9404,
                      "end": 9408
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9404,
                    "end": 9408
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "notesCount",
                    "loc": {
                      "start": 9417,
                      "end": 9427
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9417,
                    "end": 9427
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "projectsCount",
                    "loc": {
                      "start": 9436,
                      "end": 9449
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9436,
                    "end": 9449
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "questionsAskedCount",
                    "loc": {
                      "start": 9458,
                      "end": 9477
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9458,
                    "end": 9477
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "routinesCount",
                    "loc": {
                      "start": 9486,
                      "end": 9499
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9486,
                    "end": 9499
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "smartContractsCount",
                    "loc": {
                      "start": 9508,
                      "end": 9527
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9508,
                    "end": 9527
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "standardsCount",
                    "loc": {
                      "start": 9536,
                      "end": 9550
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9536,
                    "end": 9550
                  }
                },
                {
                  "kind": "Field",
                  "name": {
                    "kind": "Name",
                    "value": "theme",
                    "loc": {
                      "start": 9559,
                      "end": 9564
                    }
                  },
                  "arguments": [],
                  "directives": [],
                  "loc": {
                    "start": 9559,
                    "end": 9564
                  }
                }
              ],
              "loc": {
                "start": 417,
                "end": 9570
              }
            },
            "loc": {
              "start": 411,
              "end": 9570
            }
          }
        ],
        "loc": {
          "start": 377,
          "end": 9574
        }
      },
      "loc": {
        "start": 343,
        "end": 9574
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
    }
  },
  "rootValue": {},
  "operation": {
    "kind": "OperationDefinition",
    "operation": "mutation",
    "name": {
      "kind": "Name",
      "value": "emailResetPassword",
      "loc": {
        "start": 286,
        "end": 304
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
              "start": 306,
              "end": 311
            }
          },
          "loc": {
            "start": 305,
            "end": 311
          }
        },
        "type": {
          "kind": "NonNullType",
          "type": {
            "kind": "NamedType",
            "name": {
              "kind": "Name",
              "value": "EmailResetPasswordInput",
              "loc": {
                "start": 313,
                "end": 336
              }
            },
            "loc": {
              "start": 313,
              "end": 336
            }
          },
          "loc": {
            "start": 313,
            "end": 337
          }
        },
        "directives": [],
        "loc": {
          "start": 305,
          "end": 337
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
            "value": "emailResetPassword",
            "loc": {
              "start": 343,
              "end": 361
            }
          },
          "arguments": [
            {
              "kind": "Argument",
              "name": {
                "kind": "Name",
                "value": "input",
                "loc": {
                  "start": 362,
                  "end": 367
                }
              },
              "value": {
                "kind": "Variable",
                "name": {
                  "kind": "Name",
                  "value": "input",
                  "loc": {
                    "start": 370,
                    "end": 375
                  }
                },
                "loc": {
                  "start": 369,
                  "end": 375
                }
              },
              "loc": {
                "start": 362,
                "end": 375
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
                  "value": "isLoggedIn",
                  "loc": {
                    "start": 383,
                    "end": 393
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 383,
                  "end": 393
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "timeZone",
                  "loc": {
                    "start": 398,
                    "end": 406
                  }
                },
                "arguments": [],
                "directives": [],
                "loc": {
                  "start": 398,
                  "end": 406
                }
              },
              {
                "kind": "Field",
                "name": {
                  "kind": "Name",
                  "value": "users",
                  "loc": {
                    "start": 411,
                    "end": 416
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
                          "start": 427,
                          "end": 442
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
                                "start": 457,
                                "end": 461
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
                                      "start": 480,
                                      "end": 487
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
                                            "start": 510,
                                            "end": 512
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 510,
                                          "end": 512
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "filterType",
                                          "loc": {
                                            "start": 533,
                                            "end": 543
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 533,
                                          "end": 543
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 564,
                                            "end": 567
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
                                                  "start": 594,
                                                  "end": 596
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 594,
                                                "end": 596
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 621,
                                                  "end": 631
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 621,
                                                "end": 631
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "tag",
                                                "loc": {
                                                  "start": 656,
                                                  "end": 659
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 656,
                                                "end": 659
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "bookmarks",
                                                "loc": {
                                                  "start": 684,
                                                  "end": 693
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 684,
                                                "end": 693
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 718,
                                                  "end": 730
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
                                                        "start": 761,
                                                        "end": 763
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 761,
                                                      "end": 763
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 792,
                                                        "end": 800
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 792,
                                                      "end": 800
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 829,
                                                        "end": 840
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 829,
                                                      "end": 840
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 731,
                                                  "end": 866
                                                }
                                              },
                                              "loc": {
                                                "start": 718,
                                                "end": 866
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "you",
                                                "loc": {
                                                  "start": 891,
                                                  "end": 894
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
                                                        "start": 925,
                                                        "end": 930
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 925,
                                                      "end": 930
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isBookmarked",
                                                      "loc": {
                                                        "start": 959,
                                                        "end": 971
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 959,
                                                      "end": 971
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 895,
                                                  "end": 997
                                                }
                                              },
                                              "loc": {
                                                "start": 891,
                                                "end": 997
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 568,
                                            "end": 1019
                                          }
                                        },
                                        "loc": {
                                          "start": 564,
                                          "end": 1019
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "focusMode",
                                          "loc": {
                                            "start": 1040,
                                            "end": 1049
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
                                                  "start": 1076,
                                                  "end": 1082
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
                                                        "start": 1113,
                                                        "end": 1115
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1113,
                                                      "end": 1115
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "color",
                                                      "loc": {
                                                        "start": 1144,
                                                        "end": 1149
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1144,
                                                      "end": 1149
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "label",
                                                      "loc": {
                                                        "start": 1178,
                                                        "end": 1183
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1178,
                                                      "end": 1183
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1083,
                                                  "end": 1209
                                                }
                                              },
                                              "loc": {
                                                "start": 1076,
                                                "end": 1209
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderList",
                                                "loc": {
                                                  "start": 1234,
                                                  "end": 1246
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
                                                        "start": 1277,
                                                        "end": 1279
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1277,
                                                      "end": 1279
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 1308,
                                                        "end": 1318
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1308,
                                                      "end": 1318
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 1347,
                                                        "end": 1357
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 1347,
                                                      "end": 1357
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminders",
                                                      "loc": {
                                                        "start": 1386,
                                                        "end": 1395
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
                                                              "start": 1430,
                                                              "end": 1432
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1430,
                                                            "end": 1432
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 1465,
                                                              "end": 1475
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1465,
                                                            "end": 1475
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 1508,
                                                              "end": 1518
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1508,
                                                            "end": 1518
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 1551,
                                                              "end": 1555
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1551,
                                                            "end": 1555
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 1588,
                                                              "end": 1599
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1588,
                                                            "end": 1599
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 1632,
                                                              "end": 1639
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1632,
                                                            "end": 1639
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 1672,
                                                              "end": 1677
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1672,
                                                            "end": 1677
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 1710,
                                                              "end": 1720
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 1710,
                                                            "end": 1720
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "reminderItems",
                                                            "loc": {
                                                              "start": 1753,
                                                              "end": 1766
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
                                                                    "start": 1805,
                                                                    "end": 1807
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1805,
                                                                  "end": 1807
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "created_at",
                                                                  "loc": {
                                                                    "start": 1844,
                                                                    "end": 1854
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1844,
                                                                  "end": 1854
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "updated_at",
                                                                  "loc": {
                                                                    "start": 1891,
                                                                    "end": 1901
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1891,
                                                                  "end": 1901
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 1938,
                                                                    "end": 1942
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1938,
                                                                  "end": 1942
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 1979,
                                                                    "end": 1990
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 1979,
                                                                  "end": 1990
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "dueDate",
                                                                  "loc": {
                                                                    "start": 2027,
                                                                    "end": 2034
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2027,
                                                                  "end": 2034
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "index",
                                                                  "loc": {
                                                                    "start": 2071,
                                                                    "end": 2076
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2071,
                                                                  "end": 2076
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "isComplete",
                                                                  "loc": {
                                                                    "start": 2113,
                                                                    "end": 2123
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2113,
                                                                  "end": 2123
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 1767,
                                                              "end": 2157
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 1753,
                                                            "end": 2157
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 1396,
                                                        "end": 2187
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 1386,
                                                      "end": 2187
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 1247,
                                                  "end": 2213
                                                }
                                              },
                                              "loc": {
                                                "start": 1234,
                                                "end": 2213
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resourceList",
                                                "loc": {
                                                  "start": 2238,
                                                  "end": 2250
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
                                                        "start": 2281,
                                                        "end": 2283
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2281,
                                                      "end": 2283
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 2312,
                                                        "end": 2322
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 2312,
                                                      "end": 2322
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 2351,
                                                        "end": 2363
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
                                                              "start": 2398,
                                                              "end": 2400
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2398,
                                                            "end": 2400
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 2433,
                                                              "end": 2441
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2433,
                                                            "end": 2441
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 2474,
                                                              "end": 2485
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2474,
                                                            "end": 2485
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 2518,
                                                              "end": 2522
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2518,
                                                            "end": 2522
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2364,
                                                        "end": 2552
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2351,
                                                      "end": 2552
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "resources",
                                                      "loc": {
                                                        "start": 2581,
                                                        "end": 2590
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
                                                              "start": 2625,
                                                              "end": 2627
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2625,
                                                            "end": 2627
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 2660,
                                                              "end": 2665
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2660,
                                                            "end": 2665
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "link",
                                                            "loc": {
                                                              "start": 2698,
                                                              "end": 2702
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2698,
                                                            "end": 2702
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "usedFor",
                                                            "loc": {
                                                              "start": 2735,
                                                              "end": 2742
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 2735,
                                                            "end": 2742
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "translations",
                                                            "loc": {
                                                              "start": 2775,
                                                              "end": 2787
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
                                                                    "start": 2826,
                                                                    "end": 2828
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2826,
                                                                  "end": 2828
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "language",
                                                                  "loc": {
                                                                    "start": 2865,
                                                                    "end": 2873
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2865,
                                                                  "end": 2873
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "description",
                                                                  "loc": {
                                                                    "start": 2910,
                                                                    "end": 2921
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2910,
                                                                  "end": 2921
                                                                }
                                                              },
                                                              {
                                                                "kind": "Field",
                                                                "name": {
                                                                  "kind": "Name",
                                                                  "value": "name",
                                                                  "loc": {
                                                                    "start": 2958,
                                                                    "end": 2962
                                                                  }
                                                                },
                                                                "arguments": [],
                                                                "directives": [],
                                                                "loc": {
                                                                  "start": 2958,
                                                                  "end": 2962
                                                                }
                                                              }
                                                            ],
                                                            "loc": {
                                                              "start": 2788,
                                                              "end": 2996
                                                            }
                                                          },
                                                          "loc": {
                                                            "start": 2775,
                                                            "end": 2996
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 2591,
                                                        "end": 3026
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 2581,
                                                      "end": 3026
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 2251,
                                                  "end": 3052
                                                }
                                              },
                                              "loc": {
                                                "start": 2238,
                                                "end": 3052
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "schedule",
                                                "loc": {
                                                  "start": 3077,
                                                  "end": 3085
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
                                                        "start": 3119,
                                                        "end": 3134
                                                      }
                                                    },
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3116,
                                                      "end": 3134
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3086,
                                                  "end": 3160
                                                }
                                              },
                                              "loc": {
                                                "start": 3077,
                                                "end": 3160
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "id",
                                                "loc": {
                                                  "start": 3185,
                                                  "end": 3187
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3185,
                                                "end": 3187
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3212,
                                                  "end": 3216
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3212,
                                                "end": 3216
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3241,
                                                  "end": 3252
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3241,
                                                "end": 3252
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 1050,
                                            "end": 3274
                                          }
                                        },
                                        "loc": {
                                          "start": 1040,
                                          "end": 3274
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 488,
                                      "end": 3292
                                    }
                                  },
                                  "loc": {
                                    "start": 480,
                                    "end": 3292
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "labels",
                                    "loc": {
                                      "start": 3309,
                                      "end": 3315
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
                                            "start": 3338,
                                            "end": 3340
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3338,
                                          "end": 3340
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "color",
                                          "loc": {
                                            "start": 3361,
                                            "end": 3366
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3361,
                                          "end": 3366
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "label",
                                          "loc": {
                                            "start": 3387,
                                            "end": 3392
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3387,
                                          "end": 3392
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3316,
                                      "end": 3410
                                    }
                                  },
                                  "loc": {
                                    "start": 3309,
                                    "end": 3410
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminderList",
                                    "loc": {
                                      "start": 3427,
                                      "end": 3439
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
                                            "start": 3462,
                                            "end": 3464
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3462,
                                          "end": 3464
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 3485,
                                            "end": 3495
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3485,
                                          "end": 3495
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 3516,
                                            "end": 3526
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 3516,
                                          "end": 3526
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminders",
                                          "loc": {
                                            "start": 3547,
                                            "end": 3556
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
                                                  "start": 3583,
                                                  "end": 3585
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3583,
                                                "end": 3585
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 3610,
                                                  "end": 3620
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3610,
                                                "end": 3620
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 3645,
                                                  "end": 3655
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3645,
                                                "end": 3655
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 3680,
                                                  "end": 3684
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3680,
                                                "end": 3684
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 3709,
                                                  "end": 3720
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3709,
                                                "end": 3720
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 3745,
                                                  "end": 3752
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3745,
                                                "end": 3752
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 3777,
                                                  "end": 3782
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3777,
                                                "end": 3782
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 3807,
                                                  "end": 3817
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 3807,
                                                "end": 3817
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminderItems",
                                                "loc": {
                                                  "start": 3842,
                                                  "end": 3855
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
                                                        "start": 3886,
                                                        "end": 3888
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3886,
                                                      "end": 3888
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 3917,
                                                        "end": 3927
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3917,
                                                      "end": 3927
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 3956,
                                                        "end": 3966
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3956,
                                                      "end": 3966
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 3995,
                                                        "end": 3999
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 3995,
                                                      "end": 3999
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4028,
                                                        "end": 4039
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4028,
                                                      "end": 4039
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 4068,
                                                        "end": 4075
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4068,
                                                      "end": 4075
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 4104,
                                                        "end": 4109
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4104,
                                                      "end": 4109
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 4138,
                                                        "end": 4148
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4138,
                                                      "end": 4148
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 3856,
                                                  "end": 4174
                                                }
                                              },
                                              "loc": {
                                                "start": 3842,
                                                "end": 4174
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 3557,
                                            "end": 4196
                                          }
                                        },
                                        "loc": {
                                          "start": 3547,
                                          "end": 4196
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 3440,
                                      "end": 4214
                                    }
                                  },
                                  "loc": {
                                    "start": 3427,
                                    "end": 4214
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resourceList",
                                    "loc": {
                                      "start": 4231,
                                      "end": 4243
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
                                            "start": 4266,
                                            "end": 4268
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4266,
                                          "end": 4268
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 4289,
                                            "end": 4299
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 4289,
                                          "end": 4299
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 4320,
                                            "end": 4332
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
                                                  "start": 4359,
                                                  "end": 4361
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4359,
                                                "end": 4361
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 4386,
                                                  "end": 4394
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4386,
                                                "end": 4394
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 4419,
                                                  "end": 4430
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4419,
                                                "end": 4430
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 4455,
                                                  "end": 4459
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4455,
                                                "end": 4459
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4333,
                                            "end": 4481
                                          }
                                        },
                                        "loc": {
                                          "start": 4320,
                                          "end": 4481
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resources",
                                          "loc": {
                                            "start": 4502,
                                            "end": 4511
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
                                                  "start": 4538,
                                                  "end": 4540
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4538,
                                                "end": 4540
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 4565,
                                                  "end": 4570
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4565,
                                                "end": 4570
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "link",
                                                "loc": {
                                                  "start": 4595,
                                                  "end": 4599
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4595,
                                                "end": 4599
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "usedFor",
                                                "loc": {
                                                  "start": 4624,
                                                  "end": 4631
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 4624,
                                                "end": 4631
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 4656,
                                                  "end": 4668
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
                                                        "start": 4699,
                                                        "end": 4701
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4699,
                                                      "end": 4701
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 4730,
                                                        "end": 4738
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4730,
                                                      "end": 4738
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 4767,
                                                        "end": 4778
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4767,
                                                      "end": 4778
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 4807,
                                                        "end": 4811
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 4807,
                                                      "end": 4811
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 4669,
                                                  "end": 4837
                                                }
                                              },
                                              "loc": {
                                                "start": 4656,
                                                "end": 4837
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 4512,
                                            "end": 4859
                                          }
                                        },
                                        "loc": {
                                          "start": 4502,
                                          "end": 4859
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4244,
                                      "end": 4877
                                    }
                                  },
                                  "loc": {
                                    "start": 4231,
                                    "end": 4877
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "schedule",
                                    "loc": {
                                      "start": 4894,
                                      "end": 4902
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
                                            "start": 4928,
                                            "end": 4943
                                          }
                                        },
                                        "directives": [],
                                        "loc": {
                                          "start": 4925,
                                          "end": 4943
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 4903,
                                      "end": 4961
                                    }
                                  },
                                  "loc": {
                                    "start": 4894,
                                    "end": 4961
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "id",
                                    "loc": {
                                      "start": 4978,
                                      "end": 4980
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4978,
                                    "end": 4980
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "name",
                                    "loc": {
                                      "start": 4997,
                                      "end": 5001
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 4997,
                                    "end": 5001
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "description",
                                    "loc": {
                                      "start": 5018,
                                      "end": 5029
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5018,
                                    "end": 5029
                                  }
                                }
                              ],
                              "loc": {
                                "start": 462,
                                "end": 5043
                              }
                            },
                            "loc": {
                              "start": 457,
                              "end": 5043
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopCondition",
                              "loc": {
                                "start": 5056,
                                "end": 5069
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5056,
                              "end": 5069
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "stopTime",
                              "loc": {
                                "start": 5082,
                                "end": 5090
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5082,
                              "end": 5090
                            }
                          }
                        ],
                        "loc": {
                          "start": 443,
                          "end": 5100
                        }
                      },
                      "loc": {
                        "start": 427,
                        "end": 5100
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "apisCount",
                        "loc": {
                          "start": 5109,
                          "end": 5118
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 5109,
                        "end": 5118
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "bookmarkLists",
                        "loc": {
                          "start": 5127,
                          "end": 5140
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
                                "start": 5155,
                                "end": 5157
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5155,
                              "end": 5157
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "created_at",
                              "loc": {
                                "start": 5170,
                                "end": 5180
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5170,
                              "end": 5180
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "updated_at",
                              "loc": {
                                "start": 5193,
                                "end": 5203
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5193,
                              "end": 5203
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "label",
                              "loc": {
                                "start": 5216,
                                "end": 5221
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5216,
                              "end": 5221
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "bookmarksCount",
                              "loc": {
                                "start": 5234,
                                "end": 5248
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 5234,
                              "end": 5248
                            }
                          }
                        ],
                        "loc": {
                          "start": 5141,
                          "end": 5258
                        }
                      },
                      "loc": {
                        "start": 5127,
                        "end": 5258
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "focusModes",
                        "loc": {
                          "start": 5267,
                          "end": 5277
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
                                "start": 5292,
                                "end": 5299
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
                                      "start": 5318,
                                      "end": 5320
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5318,
                                    "end": 5320
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "filterType",
                                    "loc": {
                                      "start": 5337,
                                      "end": 5347
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 5337,
                                    "end": 5347
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "tag",
                                    "loc": {
                                      "start": 5364,
                                      "end": 5367
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
                                            "start": 5390,
                                            "end": 5392
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5390,
                                          "end": 5392
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 5413,
                                            "end": 5423
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5413,
                                          "end": 5423
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "tag",
                                          "loc": {
                                            "start": 5444,
                                            "end": 5447
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5444,
                                          "end": 5447
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "bookmarks",
                                          "loc": {
                                            "start": 5468,
                                            "end": 5477
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 5468,
                                          "end": 5477
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 5498,
                                            "end": 5510
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
                                                  "start": 5537,
                                                  "end": 5539
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5537,
                                                "end": 5539
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 5564,
                                                  "end": 5572
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5564,
                                                "end": 5572
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 5597,
                                                  "end": 5608
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5597,
                                                "end": 5608
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5511,
                                            "end": 5630
                                          }
                                        },
                                        "loc": {
                                          "start": 5498,
                                          "end": 5630
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "you",
                                          "loc": {
                                            "start": 5651,
                                            "end": 5654
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
                                                  "start": 5681,
                                                  "end": 5686
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5681,
                                                "end": 5686
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isBookmarked",
                                                "loc": {
                                                  "start": 5711,
                                                  "end": 5723
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5711,
                                                "end": 5723
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5655,
                                            "end": 5745
                                          }
                                        },
                                        "loc": {
                                          "start": 5651,
                                          "end": 5745
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5368,
                                      "end": 5763
                                    }
                                  },
                                  "loc": {
                                    "start": 5364,
                                    "end": 5763
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "focusMode",
                                    "loc": {
                                      "start": 5780,
                                      "end": 5789
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
                                            "start": 5812,
                                            "end": 5818
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
                                                  "start": 5845,
                                                  "end": 5847
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5845,
                                                "end": 5847
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "color",
                                                "loc": {
                                                  "start": 5872,
                                                  "end": 5877
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5872,
                                                "end": 5877
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "label",
                                                "loc": {
                                                  "start": 5902,
                                                  "end": 5907
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5902,
                                                "end": 5907
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5819,
                                            "end": 5929
                                          }
                                        },
                                        "loc": {
                                          "start": 5812,
                                          "end": 5929
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderList",
                                          "loc": {
                                            "start": 5950,
                                            "end": 5962
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
                                                  "start": 5989,
                                                  "end": 5991
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 5989,
                                                "end": 5991
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 6016,
                                                  "end": 6026
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6016,
                                                "end": 6026
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 6051,
                                                  "end": 6061
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6051,
                                                "end": 6061
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "reminders",
                                                "loc": {
                                                  "start": 6086,
                                                  "end": 6095
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
                                                        "start": 6126,
                                                        "end": 6128
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6126,
                                                      "end": 6128
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "created_at",
                                                      "loc": {
                                                        "start": 6157,
                                                        "end": 6167
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6157,
                                                      "end": 6167
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "updated_at",
                                                      "loc": {
                                                        "start": 6196,
                                                        "end": 6206
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6196,
                                                      "end": 6206
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 6235,
                                                        "end": 6239
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6235,
                                                      "end": 6239
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 6268,
                                                        "end": 6279
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6268,
                                                      "end": 6279
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "dueDate",
                                                      "loc": {
                                                        "start": 6308,
                                                        "end": 6315
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6308,
                                                      "end": 6315
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
                                                      "loc": {
                                                        "start": 6344,
                                                        "end": 6349
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6344,
                                                      "end": 6349
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "isComplete",
                                                      "loc": {
                                                        "start": 6378,
                                                        "end": 6388
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6378,
                                                      "end": 6388
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "reminderItems",
                                                      "loc": {
                                                        "start": 6417,
                                                        "end": 6430
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
                                                              "start": 6465,
                                                              "end": 6467
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6465,
                                                            "end": 6467
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "created_at",
                                                            "loc": {
                                                              "start": 6500,
                                                              "end": 6510
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6500,
                                                            "end": 6510
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "updated_at",
                                                            "loc": {
                                                              "start": 6543,
                                                              "end": 6553
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6543,
                                                            "end": 6553
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 6586,
                                                              "end": 6590
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6586,
                                                            "end": 6590
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 6623,
                                                              "end": 6634
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6623,
                                                            "end": 6634
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "dueDate",
                                                            "loc": {
                                                              "start": 6667,
                                                              "end": 6674
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6667,
                                                            "end": 6674
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "index",
                                                            "loc": {
                                                              "start": 6707,
                                                              "end": 6712
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6707,
                                                            "end": 6712
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "isComplete",
                                                            "loc": {
                                                              "start": 6745,
                                                              "end": 6755
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 6745,
                                                            "end": 6755
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 6431,
                                                        "end": 6785
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 6417,
                                                      "end": 6785
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6096,
                                                  "end": 6811
                                                }
                                              },
                                              "loc": {
                                                "start": 6086,
                                                "end": 6811
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 5963,
                                            "end": 6833
                                          }
                                        },
                                        "loc": {
                                          "start": 5950,
                                          "end": 6833
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "resourceList",
                                          "loc": {
                                            "start": 6854,
                                            "end": 6866
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
                                                  "start": 6893,
                                                  "end": 6895
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6893,
                                                "end": 6895
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 6920,
                                                  "end": 6930
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 6920,
                                                "end": 6930
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "translations",
                                                "loc": {
                                                  "start": 6955,
                                                  "end": 6967
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
                                                        "start": 6998,
                                                        "end": 7000
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 6998,
                                                      "end": 7000
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "language",
                                                      "loc": {
                                                        "start": 7029,
                                                        "end": 7037
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7029,
                                                      "end": 7037
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "description",
                                                      "loc": {
                                                        "start": 7066,
                                                        "end": 7077
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7066,
                                                      "end": 7077
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "name",
                                                      "loc": {
                                                        "start": 7106,
                                                        "end": 7110
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7106,
                                                      "end": 7110
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 6968,
                                                  "end": 7136
                                                }
                                              },
                                              "loc": {
                                                "start": 6955,
                                                "end": 7136
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "resources",
                                                "loc": {
                                                  "start": 7161,
                                                  "end": 7170
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
                                                        "start": 7201,
                                                        "end": 7203
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7201,
                                                      "end": 7203
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "index",
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
                                                      "value": "link",
                                                      "loc": {
                                                        "start": 7266,
                                                        "end": 7270
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7266,
                                                      "end": 7270
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "usedFor",
                                                      "loc": {
                                                        "start": 7299,
                                                        "end": 7306
                                                      }
                                                    },
                                                    "arguments": [],
                                                    "directives": [],
                                                    "loc": {
                                                      "start": 7299,
                                                      "end": 7306
                                                    }
                                                  },
                                                  {
                                                    "kind": "Field",
                                                    "name": {
                                                      "kind": "Name",
                                                      "value": "translations",
                                                      "loc": {
                                                        "start": 7335,
                                                        "end": 7347
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
                                                              "start": 7382,
                                                              "end": 7384
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7382,
                                                            "end": 7384
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "language",
                                                            "loc": {
                                                              "start": 7417,
                                                              "end": 7425
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7417,
                                                            "end": 7425
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "description",
                                                            "loc": {
                                                              "start": 7458,
                                                              "end": 7469
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7458,
                                                            "end": 7469
                                                          }
                                                        },
                                                        {
                                                          "kind": "Field",
                                                          "name": {
                                                            "kind": "Name",
                                                            "value": "name",
                                                            "loc": {
                                                              "start": 7502,
                                                              "end": 7506
                                                            }
                                                          },
                                                          "arguments": [],
                                                          "directives": [],
                                                          "loc": {
                                                            "start": 7502,
                                                            "end": 7506
                                                          }
                                                        }
                                                      ],
                                                      "loc": {
                                                        "start": 7348,
                                                        "end": 7536
                                                      }
                                                    },
                                                    "loc": {
                                                      "start": 7335,
                                                      "end": 7536
                                                    }
                                                  }
                                                ],
                                                "loc": {
                                                  "start": 7171,
                                                  "end": 7562
                                                }
                                              },
                                              "loc": {
                                                "start": 7161,
                                                "end": 7562
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 6867,
                                            "end": 7584
                                          }
                                        },
                                        "loc": {
                                          "start": 6854,
                                          "end": 7584
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "schedule",
                                          "loc": {
                                            "start": 7605,
                                            "end": 7613
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
                                                  "start": 7643,
                                                  "end": 7658
                                                }
                                              },
                                              "directives": [],
                                              "loc": {
                                                "start": 7640,
                                                "end": 7658
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 7614,
                                            "end": 7680
                                          }
                                        },
                                        "loc": {
                                          "start": 7605,
                                          "end": 7680
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "id",
                                          "loc": {
                                            "start": 7701,
                                            "end": 7703
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7701,
                                          "end": 7703
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 7724,
                                            "end": 7728
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7724,
                                          "end": 7728
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 7749,
                                            "end": 7760
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 7749,
                                          "end": 7760
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 5790,
                                      "end": 7778
                                    }
                                  },
                                  "loc": {
                                    "start": 5780,
                                    "end": 7778
                                  }
                                }
                              ],
                              "loc": {
                                "start": 5300,
                                "end": 7792
                              }
                            },
                            "loc": {
                              "start": 5292,
                              "end": 7792
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "labels",
                              "loc": {
                                "start": 7805,
                                "end": 7811
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
                                      "start": 7830,
                                      "end": 7832
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7830,
                                    "end": 7832
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "color",
                                    "loc": {
                                      "start": 7849,
                                      "end": 7854
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7849,
                                    "end": 7854
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "label",
                                    "loc": {
                                      "start": 7871,
                                      "end": 7876
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7871,
                                    "end": 7876
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7812,
                                "end": 7890
                              }
                            },
                            "loc": {
                              "start": 7805,
                              "end": 7890
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "reminderList",
                              "loc": {
                                "start": 7903,
                                "end": 7915
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
                                      "start": 7934,
                                      "end": 7936
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7934,
                                    "end": 7936
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 7953,
                                      "end": 7963
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7953,
                                    "end": 7963
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "updated_at",
                                    "loc": {
                                      "start": 7980,
                                      "end": 7990
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 7980,
                                    "end": 7990
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "reminders",
                                    "loc": {
                                      "start": 8007,
                                      "end": 8016
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
                                            "start": 8039,
                                            "end": 8041
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8039,
                                          "end": 8041
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "created_at",
                                          "loc": {
                                            "start": 8062,
                                            "end": 8072
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8062,
                                          "end": 8072
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "updated_at",
                                          "loc": {
                                            "start": 8093,
                                            "end": 8103
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8093,
                                          "end": 8103
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8124,
                                            "end": 8128
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8124,
                                          "end": 8128
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8149,
                                            "end": 8160
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8149,
                                          "end": 8160
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "dueDate",
                                          "loc": {
                                            "start": 8181,
                                            "end": 8188
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8181,
                                          "end": 8188
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8209,
                                            "end": 8214
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8209,
                                          "end": 8214
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "isComplete",
                                          "loc": {
                                            "start": 8235,
                                            "end": 8245
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8235,
                                          "end": 8245
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "reminderItems",
                                          "loc": {
                                            "start": 8266,
                                            "end": 8279
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
                                                  "start": 8306,
                                                  "end": 8308
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8306,
                                                "end": 8308
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "created_at",
                                                "loc": {
                                                  "start": 8333,
                                                  "end": 8343
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8333,
                                                "end": 8343
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "updated_at",
                                                "loc": {
                                                  "start": 8368,
                                                  "end": 8378
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8368,
                                                "end": 8378
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 8403,
                                                  "end": 8407
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8403,
                                                "end": 8407
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 8432,
                                                  "end": 8443
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8432,
                                                "end": 8443
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "dueDate",
                                                "loc": {
                                                  "start": 8468,
                                                  "end": 8475
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8468,
                                                "end": 8475
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "index",
                                                "loc": {
                                                  "start": 8500,
                                                  "end": 8505
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8500,
                                                "end": 8505
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "isComplete",
                                                "loc": {
                                                  "start": 8530,
                                                  "end": 8540
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 8530,
                                                "end": 8540
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8280,
                                            "end": 8562
                                          }
                                        },
                                        "loc": {
                                          "start": 8266,
                                          "end": 8562
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8017,
                                      "end": 8580
                                    }
                                  },
                                  "loc": {
                                    "start": 8007,
                                    "end": 8580
                                  }
                                }
                              ],
                              "loc": {
                                "start": 7916,
                                "end": 8594
                              }
                            },
                            "loc": {
                              "start": 7903,
                              "end": 8594
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "resourceList",
                              "loc": {
                                "start": 8607,
                                "end": 8619
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
                                      "start": 8638,
                                      "end": 8640
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8638,
                                    "end": 8640
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "created_at",
                                    "loc": {
                                      "start": 8657,
                                      "end": 8667
                                    }
                                  },
                                  "arguments": [],
                                  "directives": [],
                                  "loc": {
                                    "start": 8657,
                                    "end": 8667
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "translations",
                                    "loc": {
                                      "start": 8684,
                                      "end": 8696
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
                                            "start": 8719,
                                            "end": 8721
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8719,
                                          "end": 8721
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "language",
                                          "loc": {
                                            "start": 8742,
                                            "end": 8750
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8742,
                                          "end": 8750
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "description",
                                          "loc": {
                                            "start": 8771,
                                            "end": 8782
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8771,
                                          "end": 8782
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "name",
                                          "loc": {
                                            "start": 8803,
                                            "end": 8807
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8803,
                                          "end": 8807
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8697,
                                      "end": 8825
                                    }
                                  },
                                  "loc": {
                                    "start": 8684,
                                    "end": 8825
                                  }
                                },
                                {
                                  "kind": "Field",
                                  "name": {
                                    "kind": "Name",
                                    "value": "resources",
                                    "loc": {
                                      "start": 8842,
                                      "end": 8851
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
                                            "start": 8874,
                                            "end": 8876
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8874,
                                          "end": 8876
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "index",
                                          "loc": {
                                            "start": 8897,
                                            "end": 8902
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8897,
                                          "end": 8902
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "link",
                                          "loc": {
                                            "start": 8923,
                                            "end": 8927
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8923,
                                          "end": 8927
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "usedFor",
                                          "loc": {
                                            "start": 8948,
                                            "end": 8955
                                          }
                                        },
                                        "arguments": [],
                                        "directives": [],
                                        "loc": {
                                          "start": 8948,
                                          "end": 8955
                                        }
                                      },
                                      {
                                        "kind": "Field",
                                        "name": {
                                          "kind": "Name",
                                          "value": "translations",
                                          "loc": {
                                            "start": 8976,
                                            "end": 8988
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
                                                  "start": 9015,
                                                  "end": 9017
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9015,
                                                "end": 9017
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "language",
                                                "loc": {
                                                  "start": 9042,
                                                  "end": 9050
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9042,
                                                "end": 9050
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "description",
                                                "loc": {
                                                  "start": 9075,
                                                  "end": 9086
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9075,
                                                "end": 9086
                                              }
                                            },
                                            {
                                              "kind": "Field",
                                              "name": {
                                                "kind": "Name",
                                                "value": "name",
                                                "loc": {
                                                  "start": 9111,
                                                  "end": 9115
                                                }
                                              },
                                              "arguments": [],
                                              "directives": [],
                                              "loc": {
                                                "start": 9111,
                                                "end": 9115
                                              }
                                            }
                                          ],
                                          "loc": {
                                            "start": 8989,
                                            "end": 9137
                                          }
                                        },
                                        "loc": {
                                          "start": 8976,
                                          "end": 9137
                                        }
                                      }
                                    ],
                                    "loc": {
                                      "start": 8852,
                                      "end": 9155
                                    }
                                  },
                                  "loc": {
                                    "start": 8842,
                                    "end": 9155
                                  }
                                }
                              ],
                              "loc": {
                                "start": 8620,
                                "end": 9169
                              }
                            },
                            "loc": {
                              "start": 8607,
                              "end": 9169
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "schedule",
                              "loc": {
                                "start": 9182,
                                "end": 9190
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
                                      "start": 9212,
                                      "end": 9227
                                    }
                                  },
                                  "directives": [],
                                  "loc": {
                                    "start": 9209,
                                    "end": 9227
                                  }
                                }
                              ],
                              "loc": {
                                "start": 9191,
                                "end": 9241
                              }
                            },
                            "loc": {
                              "start": 9182,
                              "end": 9241
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "id",
                              "loc": {
                                "start": 9254,
                                "end": 9256
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9254,
                              "end": 9256
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "name",
                              "loc": {
                                "start": 9269,
                                "end": 9273
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9269,
                              "end": 9273
                            }
                          },
                          {
                            "kind": "Field",
                            "name": {
                              "kind": "Name",
                              "value": "description",
                              "loc": {
                                "start": 9286,
                                "end": 9297
                              }
                            },
                            "arguments": [],
                            "directives": [],
                            "loc": {
                              "start": 9286,
                              "end": 9297
                            }
                          }
                        ],
                        "loc": {
                          "start": 5278,
                          "end": 9307
                        }
                      },
                      "loc": {
                        "start": 5267,
                        "end": 9307
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "handle",
                        "loc": {
                          "start": 9316,
                          "end": 9322
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9316,
                        "end": 9322
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "hasPremium",
                        "loc": {
                          "start": 9331,
                          "end": 9341
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9331,
                        "end": 9341
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "id",
                        "loc": {
                          "start": 9350,
                          "end": 9352
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9350,
                        "end": 9352
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "languages",
                        "loc": {
                          "start": 9361,
                          "end": 9370
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9361,
                        "end": 9370
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "membershipsCount",
                        "loc": {
                          "start": 9379,
                          "end": 9395
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9379,
                        "end": 9395
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "name",
                        "loc": {
                          "start": 9404,
                          "end": 9408
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9404,
                        "end": 9408
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "notesCount",
                        "loc": {
                          "start": 9417,
                          "end": 9427
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9417,
                        "end": 9427
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "projectsCount",
                        "loc": {
                          "start": 9436,
                          "end": 9449
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9436,
                        "end": 9449
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "questionsAskedCount",
                        "loc": {
                          "start": 9458,
                          "end": 9477
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9458,
                        "end": 9477
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "routinesCount",
                        "loc": {
                          "start": 9486,
                          "end": 9499
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9486,
                        "end": 9499
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "smartContractsCount",
                        "loc": {
                          "start": 9508,
                          "end": 9527
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9508,
                        "end": 9527
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "standardsCount",
                        "loc": {
                          "start": 9536,
                          "end": 9550
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9536,
                        "end": 9550
                      }
                    },
                    {
                      "kind": "Field",
                      "name": {
                        "kind": "Name",
                        "value": "theme",
                        "loc": {
                          "start": 9559,
                          "end": 9564
                        }
                      },
                      "arguments": [],
                      "directives": [],
                      "loc": {
                        "start": 9559,
                        "end": 9564
                      }
                    }
                  ],
                  "loc": {
                    "start": 417,
                    "end": 9570
                  }
                },
                "loc": {
                  "start": 411,
                  "end": 9570
                }
              }
            ],
            "loc": {
              "start": 377,
              "end": 9574
            }
          },
          "loc": {
            "start": 343,
            "end": 9574
          }
        }
      ],
      "loc": {
        "start": 339,
        "end": 9576
      }
    },
    "loc": {
      "start": 277,
      "end": 9576
    }
  },
  "variableValues": {},
  "path": {
    "key": "auth_emailResetPassword"
  }
} as const;
